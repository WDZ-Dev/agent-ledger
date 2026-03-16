# Stage 1: Build
FROM golang:1.25-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /agentledger ./cmd/agentledger

# Stage 2: Runtime
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /agentledger /agentledger
COPY configs/agentledger.example.yaml /etc/agentledger/agentledger.yaml

EXPOSE 8787
VOLUME /data

ENTRYPOINT ["/agentledger"]
CMD ["serve"]
