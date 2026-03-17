# Installation

## Homebrew

```bash
brew install wdz-dev/tap/agentledger
```

## Binary Download

Download the latest release from [GitHub Releases](https://github.com/WDZ-Dev/agent-ledger/releases):

```bash
curl -sSL https://github.com/WDZ-Dev/agent-ledger/releases/latest/download/agentledger_$(uname -s)_$(uname -m).tar.gz | tar xz
sudo mv agentledger /usr/local/bin/
```

Available for Linux, macOS, and Windows on both amd64 and arm64.

## Docker

```bash
docker run --rm -p 8787:8787 ghcr.io/wdz-dev/agent-ledger:latest
```

See [Docker deployment](../deployment/docker.md) for persistence and configuration.

## Helm (Kubernetes)

```bash
helm install agentledger deploy/helm/agentledger
```

See [Kubernetes deployment](../deployment/kubernetes.md) for full details.

## From Source

Requires Go 1.25+:

```bash
go install github.com/WDZ-Dev/agent-ledger/cmd/agentledger@latest
```

## Verify

```bash
agentledger version
```
