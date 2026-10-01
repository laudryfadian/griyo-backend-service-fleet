FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /service ./cmd/service
FROM alpine:3.22
RUN apk add --no-cache ca-certificates git && adduser -D -u 10001 griyo
COPY --from=build /service /service
USER griyo
ENTRYPOINT ["/service"]
