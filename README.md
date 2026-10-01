# griyo-backend-service-fleet

React dashboard and Go services for an AI office spanning multiple VPS servers.

## Repositories

Clone all six repositories into a common directory:

- griyo-frontend: React, TypeScript strict interfaces, runtime Zod API validation.
- griyo-backend-service-gateway: owner HTTP API, runner HTTP API, Swagger and centralized local environment/Compose.
- griyo-backend-service-workspace: projects, agents, knowledge, workspace settings.
- griyo-backend-service-fleet: per-VPS model configuration, one-time pairing token, runner heartbeat.
- griyo-backend-service-task: queue, concurrency, leases, budget reservation, review and approval records.
- griyo-backend-service-runner: outbound HTTPS polling from each VPS; model provider integration and isolated proposal directory.

## Run locally

Requirements: Docker Compose, Python 3 (credential generator). All repositories must be siblings.

```sh
cd griyo-backend-service-gateway/deploy
./generate-env.sh
docker compose --env-file .env up --build -d
```

Open http://localhost:3000. Sign in with `OWNER_PASSWORD` from the generated `.env` (never commit that file). Swagger is http://localhost:3000/api/docs after login. PostgreSQL 16 persists in the local Docker volume `griyo_postgresData`. No hosted database is used. Schemas are owned separately by workspace, fleet and task within the same local PostgreSQL instance; this keeps local setup simple.

Create a VPS in Infrastructure, set a model identifier supported by your provider, its OpenAI-compatible `/v1` base URL and credential reference (e.g. `AI_API_KEY`). Pair runner, copy the one-time token to `RUNNER_TOKEN` in the centralized `.env`. Set `AI_API_KEY` and positive `AI_INPUT_USD_PER_MILLION`/`AI_OUTPUT_USD_PER_MILLION` for that runner's selected model. Restart runner after environment changes:

```sh
docker compose --env-file .env --profile runner up --build -d runner
```

The central `.env` belongs to the Griyo control plane. An external VPS needs its own runner credentials (unique pairing token and provider keys); never distribute the central database or session secrets to remote VPS machines. Remote runners require the control plane behind an HTTPS reverse proxy and `GRIYO_URL` set to its HTTPS URL. Set `APP_ORIGIN` to that URL and `COOKIE_SECURE=true`. Compose binds development ports to localhost by default.

After the runner shows Online: create a project mapped to work/deploy VPS, assign one agent, create a task. An agent can have one nullable `projectId`; unassigned agents appear as “Belum di project”. Runner retrieves an optional public HTTPS repository, reads bounded source context, asks the model for a JSON file proposal, saves files to `/workspaces/{taskId}` and reports the proposal for review. The queue reserves maximum task budgets and caps per-VPS concurrency. Runner renews a 90-second lease every 20 seconds. Expired leases fail rather than automatically duplicating an AI call.

## Current execution boundary

This version supports real persisted management and model-generated code proposals. It does **not** execute repository scripts, create GitHub pull requests, push changes to project repositories, or deploy applications. Tests shown in the proposal are suggested checks, not executed results. Approval is an owner decision record, not a deployment success signal. Container command endpoints are reserved for a future execution adapter and are not offered as working UI actions. Private repository authentication, per-task sandbox containers, automatic test execution, deployment adapters, fallback models and retries still require implementation. Project knowledge is snapshotted into instructions when a task is created. Workspace daily/monthly settings are informational; enforced execution reservation is the VPS daily budget. Costs are estimates computed from provider token usage and configured rates, not billing reconciliation.

This is a development implementation, not a production certification. gRPC uses service-token metadata on the private Compose network; use mTLS if those ports are exposed beyond that network. Provider keys stay on the runner. Fleet stores a hash of its pairing token. Owner authentication uses an HTTP-only session cookie; it is a single-owner application.

## Development and verification

Frontend: `npm ci && npm run build && npm test` (strict TypeScript, no explicit `any`). Services: `go test ./... && go vet ./... && go build ./cmd/service`. Each repository includes CI. Go exported identifiers use uppercase to satisfy the language's visibility rules; JSON fields and internal variables use camelCase. Generated protobuf code follows generator conventions.

HTTP resources: `/api/projects`, `/api/agents`, `/api/servers`, `/api/tasks`, `/api/documents`, `/api/settings`, `/api/approvals`, `/api/events`; nested task messages/approval and server pairing. Services communicate over generated gRPC contracts in `proto/griyo.proto`; the transport envelope carries strict domain JSON structs. All internal operations require `SERVICE_TOKEN`. PostgreSQL tables are initialized on service startup.

The frontend reuses the prototype's stylesheet and visual foundation, with real empty states and live data. Full screenshot parity with every prototype interaction has not been certified.
