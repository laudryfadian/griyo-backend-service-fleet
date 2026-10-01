package internal

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/laudryfadian/griyo-backend-service-fleet/internal/domain"
	"github.com/laudryfadian/griyo-backend-service-fleet/internal/pb"
	"github.com/laudryfadian/griyo-backend-service-fleet/internal/platform"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/url"
	"time"
)

const migration = `CREATE SCHEMA IF NOT EXISTS fleet;
CREATE TABLE IF NOT EXISTS fleet.servers(id text PRIMARY KEY,data jsonb NOT NULL,last_seen timestamptz,token_hash text UNIQUE);
CREATE TABLE IF NOT EXISTS fleet.commands(id text PRIMARY KEY,server_id text NOT NULL REFERENCES fleet.servers(id),data jsonb NOT NULL,status text NOT NULL DEFAULT 'Queued');`

type Service struct {
	pb.UnimplementedFleetServiceServer
	db *pgxpool.Pool
}

func New(ctx context.Context, db *pgxpool.Pool) (*Service, error) {
	_, e := db.Exec(ctx, migration)
	return &Service{db: db}, e
}
func ValidateConfig(c domain.ServerConfig) error {
	if c.Model == "" || c.CredentialRef == "" {
		return platform.Invalid("model and credentialRef are required")
	}
	u, e := url.Parse(c.Endpoint)
	if e != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return platform.Invalid("valid AI endpoint required")
	}
	if c.DailyBudget <= 0 || c.MaxConcurrency < 1 || c.MaxConcurrency > 20 || c.TimeoutMinutes < 1 || c.TimeoutMinutes > 240 || c.MaxRetries < 0 || c.MaxRetries > 5 {
		return platform.Invalid("invalid execution limits")
	}
	return nil
}
func (s *Service) server(ctx context.Context, id string) (domain.Server, error) {
	var v domain.Server
	var body []byte
	var seen *time.Time
	e := s.db.QueryRow(ctx, "SELECT data,last_seen FROM fleet.servers WHERE id=$1", id).Scan(&body, &seen)
	if errors.Is(e, pgx.ErrNoRows) {
		return v, platform.NotFound()
	}
	if e != nil {
		return v, platform.Internal(e)
	}
	if e = json.Unmarshal(body, &v); e != nil {
		return v, platform.Internal(e)
	}
	v.LastSeen = seen
	v.Status = "Offline"
	if seen != nil && time.Since(*seen) < 60*time.Second {
		v.Status = "Online"
	}
	if v.Metrics.Containers == nil {
		v.Metrics.Containers = []domain.Container{}
	}
	return v, nil
}
func (s *Service) Execute(ctx context.Context, r *pb.Request) (*pb.Response, error) {
	if r.Operation == "authenticate" {
		var h domain.TokenHash
		if e := platform.Decode(r.Body, &h); e != nil {
			return nil, e
		}
		var id string
		e := s.db.QueryRow(ctx, "SELECT id FROM fleet.servers WHERE token_hash=$1", h.TokenHash).Scan(&id)
		if e != nil {
			return nil, status.Error(codes.Unauthenticated, "runner token invalid or revoked")
		}
		return platform.Reply(domain.RunnerIdentity{ServerId: id})
	}
	if r.Resource == "commands" {
		return s.commands(ctx, r)
	}
	if r.Resource != "servers" {
		return nil, platform.Invalid("unknown resource")
	}
	switch r.Operation {
	case "list":
		rows, e := s.db.Query(ctx, "SELECT id FROM fleet.servers ORDER BY id")
		if e != nil {
			return nil, platform.Internal(e)
		}
		defer rows.Close()
		ids := []string{}
		for rows.Next() {
			var id string
			if e = rows.Scan(&id); e != nil {
				return nil, platform.Internal(e)
			}
			ids = append(ids, id)
		}
		if e = rows.Err(); e != nil {
			return nil, platform.Internal(e)
		}
		rows.Close()
		items := []domain.Server{}
		for _, id := range ids {
			v, e := s.server(ctx, id)
			if e != nil {
				return nil, e
			}
			items = append(items, v)
		}
		return platform.Reply(items)
	case "get":
		v, e := s.server(ctx, r.Id)
		if e != nil {
			return nil, e
		}
		return platform.Reply(v)
	case "pair":
		token := platform.Id() + platform.Id()
		tag, e := s.db.Exec(ctx, "UPDATE fleet.servers SET token_hash=$2 WHERE id=$1", r.Id, platform.Hash(token))
		if e != nil {
			return nil, platform.Internal(e)
		}
		if tag.RowsAffected() == 0 {
			return nil, platform.NotFound()
		}
		return platform.Reply(domain.PairToken{ServerId: r.Id, Token: token})
	case "revoke":
		tag, e := s.db.Exec(ctx, "UPDATE fleet.servers SET token_hash=NULL,last_seen=NULL WHERE id=$1", r.Id)
		if e != nil {
			return nil, platform.Internal(e)
		}
		if tag.RowsAffected() == 0 {
			return nil, platform.NotFound()
		}
		return platform.Reply(domain.RunnerIdentity{ServerId: r.Id})
	case "heartbeat":
		var metrics domain.Metrics
		if e := platform.Decode(r.Body, &metrics); e != nil {
			return nil, e
		}
		v, e := s.server(ctx, r.Id)
		if e != nil {
			return nil, e
		}
		if metrics.CpuPercent < 0 || metrics.CpuPercent > 100 {
			return nil, platform.Invalid("invalid cpuPercent")
		}
		v.Metrics = metrics
		body, _ := json.Marshal(v)
		_, e = s.db.Exec(ctx, "UPDATE fleet.servers SET data=$2,last_seen=now() WHERE id=$1", r.Id, body)
		if e != nil {
			return nil, platform.Internal(e)
		}
		return platform.Reply(domain.RunnerIdentity{ServerId: r.Id})
	case "create", "update":
		var v domain.Server
		if e := platform.Decode(r.Body, &v); e != nil {
			return nil, e
		}
		if e := platform.ValidateName(v.Name); e != nil {
			return nil, e
		}
		if v.Role != "Office" && v.Role != "Development" && v.Role != "Staging" && v.Role != "Production" {
			return nil, platform.Invalid("invalid server role")
		}
		if e := ValidateConfig(v.Config); e != nil {
			return nil, e
		}
		if v.Role == "Production" {
			v.Config.RequireApproval = true
		}
		v.Status = "Offline"
		v.Id = r.Id
		if r.Operation == "create" {
			v.Id = platform.Id()
		}
		if v.Config.ModelOverrides == nil {
			v.Config.ModelOverrides = map[string]string{}
		}
		if r.Operation == "update" {
			old, e := s.server(ctx, r.Id)
			if e != nil {
				return nil, e
			}
			v.Metrics = old.Metrics
			v.LastSeen = old.LastSeen
		}
		body, _ := json.Marshal(v)
		query := "INSERT INTO fleet.servers(id,data) VALUES($1,$2)"
		if r.Operation == "update" {
			query = "UPDATE fleet.servers SET data=$2 WHERE id=$1"
		}
		tag, e := s.db.Exec(ctx, query, v.Id, body)
		if e != nil {
			return nil, platform.Internal(e)
		}
		if tag.RowsAffected() == 0 {
			return nil, platform.NotFound()
		}
		return platform.Reply(v)
	default:
		return nil, platform.Invalid("unsupported operation")
	}
}
func (s *Service) commands(ctx context.Context, r *pb.Request) (*pb.Response, error) {
	switch r.Operation {
	case "create":
		var c domain.ContainerCommand
		if e := platform.Decode(r.Body, &c); e != nil {
			return nil, e
		}
		if c.Action != "restart" && c.Action != "start" && c.Action != "stop" && c.Action != "logs" {
			return nil, platform.Invalid("invalid container action")
		}
		v, e := s.server(ctx, c.ServerId)
		if e != nil {
			return nil, e
		}
		if v.Status != "Online" {
			return nil, platform.Conflict("server offline")
		}
		found := false
		for _, container := range v.Metrics.Containers {
			if container.Id == c.ContainerId {
				found = true
			}
		}
		if !found {
			return nil, platform.Invalid("container not reported by runner")
		}
		c.Id = platform.Id()
		c.Status = "Queued"
		c.CreatedAt = time.Now().UTC()
		body, _ := json.Marshal(c)
		_, e = s.db.Exec(ctx, "INSERT INTO fleet.commands(id,server_id,data) VALUES($1,$2,$3)", c.Id, c.ServerId, body)
		if e != nil {
			return nil, platform.Internal(e)
		}
		return platform.Reply(c)
	case "list":
		rows, e := s.db.Query(ctx, "SELECT data FROM fleet.commands WHERE server_id=$1 ORDER BY id", r.Id)
		if e != nil {
			return nil, platform.Internal(e)
		}
		defer rows.Close()
		items := []json.RawMessage{}
		for rows.Next() {
			var b json.RawMessage
			if e = rows.Scan(&b); e != nil {
				return nil, platform.Internal(e)
			}
			items = append(items, b)
		}
		return platform.Reply(items)
	case "claim":
		var b []byte
		e := s.db.QueryRow(ctx, `UPDATE fleet.commands SET status='Working',data=jsonb_set(data,'{status}','"Working"') WHERE id=(SELECT id FROM fleet.commands WHERE server_id=$1 AND status='Queued' ORDER BY id FOR UPDATE SKIP LOCKED LIMIT 1) RETURNING data`, r.Id).Scan(&b)
		if errors.Is(e, pgx.ErrNoRows) {
			return &pb.Response{Body: []byte("null")}, nil
		}
		if e != nil {
			return nil, platform.Internal(e)
		}
		return &pb.Response{Body: b}, nil
	case "report":
		var c domain.ContainerCommand
		if e := platform.Decode(r.Body, &c); e != nil {
			return nil, e
		}
		if c.Status != "Done" && c.Status != "Failed" {
			return nil, platform.Invalid("invalid report status")
		}
		body, _ := json.Marshal(c)
		tag, e := s.db.Exec(ctx, "UPDATE fleet.commands SET data=$3,status=$4 WHERE id=$1 AND server_id=$2 AND status='Working'", c.Id, r.Id, body, c.Status)
		if e != nil {
			return nil, platform.Internal(e)
		}
		if tag.RowsAffected() == 0 {
			return nil, platform.NotFound()
		}
		return platform.Reply(c)
	}
	return nil, platform.Invalid("unsupported command operation")
}
