package platform

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/laudryfadian/griyo-backend-service-fleet/internal/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func Env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func Require(key string) string {
	v := os.Getenv(key)
	if len(v) < 24 {
		panic(key + " must contain at least 24 characters")
	}
	return v
}
func Id() string {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}
func Hash(value string) string { h := sha256.Sum256([]byte(value)); return hex.EncodeToString(h[:]) }
func Equal(a, b string) bool   { return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 }
func Decode(body []byte, value interface{}) error {
	d := json.NewDecoder(strings.NewReader(string(body)))
	d.DisallowUnknownFields()
	if e := d.Decode(value); e != nil {
		return status.Error(codes.InvalidArgument, "invalid request body: "+e.Error())
	}
	if e := d.Decode(new(json.RawMessage)); e != io.EOF {
		return status.Error(codes.InvalidArgument, "unexpected trailing JSON")
	}
	return nil
}
func Reply(value interface{}) (*pb.Response, error) {
	b, e := json.Marshal(value)
	if e != nil {
		return nil, status.Error(codes.Internal, "cannot encode response")
	}
	return &pb.Response{Body: b}, nil
}
func Db(ctx context.Context) (*pgxpool.Pool, error) {
	pool, e := pgxpool.New(ctx, Env("DATABASE_URL", ""))
	if e != nil {
		return nil, e
	}
	for i := 0; i < 30; i++ {
		if e = pool.Ping(ctx); e == nil {
			return pool, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	pool.Close()
	return nil, e
}
func Dial(addr string) (*grpc.ClientConn, error) {
	return grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply interface{}, conn *grpc.ClientConn, next grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-service-token", Require("SERVICE_TOKEN"))
		return next(ctx, method, req, reply, conn, opts...)
	}))
}
func Server() *grpc.Server {
	token := Require("SERVICE_TOKEN")
	return grpc.NewServer(grpc.MaxRecvMsgSize(2<<20), grpc.UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, _ := metadata.FromIncomingContext(ctx)
		values := md.Get("x-service-token")
		if len(values) != 1 || !Equal(values[0], token) {
			return nil, status.Error(codes.Unauthenticated, "invalid service token")
		}
		return handler(ctx, req)
	}))
}
func ServeGrpc(s *grpc.Server, addr string) error {
	listener, e := net.Listen("tcp", addr)
	if e != nil {
		return e
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	go func() {
		<-ctx.Done()
		done := make(chan struct{})
		go func() { s.GracefulStop(); close(done) }()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			s.Stop()
		}
	}()
	return s.Serve(listener)
}
func HttpError(w http.ResponseWriter, e error) {
	code := http.StatusInternalServerError
	switch status.Code(e) {
	case codes.InvalidArgument:
		code = 400
	case codes.NotFound:
		code = 404
	case codes.AlreadyExists, codes.FailedPrecondition, codes.Aborted:
		code = 409
	case codes.Unauthenticated:
		code = 401
	case codes.PermissionDenied:
		code = 403
	case codes.ResourceExhausted:
		code = 429
	case codes.Unavailable:
		code = 503
	case codes.DeadlineExceeded:
		code = 504
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{status.Convert(e).Message()})
}
func NotFound() error            { return status.Error(codes.NotFound, "resource not found") }
func Invalid(text string) error  { return status.Error(codes.InvalidArgument, text) }
func Conflict(text string) error { return status.Error(codes.FailedPrecondition, text) }
func Internal(e error) error {
	if e == nil {
		return nil
	}
	return status.Error(codes.Internal, "database operation failed")
}
func ValidateName(name string) error {
	if strings.TrimSpace(name) == "" || len(name) > 120 {
		return Invalid("name is required and must be under 120 characters")
	}
	return nil
}
func Request(operation, resource, id string, body []byte) *pb.Request {
	return &pb.Request{Operation: operation, Resource: resource, Id: id, Body: body}
}
func Context() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

var _ = errors.New
var _ = fmt.Sprintf
