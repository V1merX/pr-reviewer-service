ARG GO_VERSION=1.24

FROM golang:${GO_VERSION}-alpine AS build-stage

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/pr-reviewer ./cmd/api/main.go

FROM alpine:latest AS run-stage
RUN apk add --no-cache ca-certificates

WORKDIR /home/app

COPY --from=build-stage /out/pr-reviewer ./pr-reviewer
COPY --from=build-stage /src/config ./config

RUN adduser -D appuser && chown -R appuser:appuser /home/app
USER appuser

EXPOSE 8080

ENTRYPOINT ["./pr-reviewer"]