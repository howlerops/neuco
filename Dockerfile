FROM golang:1.25-alpine AS base
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

FROM base AS server-build
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/neuco-api ./cmd/server

FROM base AS worker-build
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/neuco-worker ./cmd/worker

FROM alpine:3.19 AS server
RUN apk add --no-cache ca-certificates tzdata
COPY --from=server-build /bin/neuco-api /bin/neuco-api
EXPOSE 8080
ENTRYPOINT ["/bin/neuco-api"]

FROM alpine:3.19 AS worker
RUN apk add --no-cache ca-certificates tzdata
COPY --from=worker-build /bin/neuco-worker /bin/neuco-worker
ENTRYPOINT ["/bin/neuco-worker"]
