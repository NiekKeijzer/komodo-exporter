# komodo-exporter

[![CI](https://github.com/NiekKeijzer/komodo-exporter/actions/workflows/build.yml/badge.svg)](https://github.com/NiekKeijzer/komodo-exporter/actions)
[![AI:X I curate](https://img.shields.io/badge/-I%20curate-grey?style=for-the-badge&logo=data%3Aimage%2Fsvg%2Bxml%3Bbase64%2CPHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAxMDAwIDEwMDAiIHdpZHRoPSIxMDAwIiBoZWlnaHQ9IjEwMDAiIGZpbGw9IndoaXRlIj4KICAgIDxjaXJjbGUgY3g9IjUwMCIgY3k9IjUwMCIgcj0iNDUyLjMiIGZpbGw9Im5vbmUiIHN0cm9rZT0id2hpdGUiIHN0cm9rZS13aWR0aD0iNDguNyIgLz4KICAgIDxwYXRoIGQ9Ik0yMzkuMSA0MTUuNkwyMTQuNTMgNTIzLjZMMjYzLjg1IDUyMy42TDIzOS4xIDQxNS42Wk0yMDcuMjkgMzY3Ljg4TDI3MS4xIDM2Ny44OEwzNDIuMzMgNjMxLjc4TDI5MC4xOSA2MzEuNzhMMjczLjkyIDU2Ni41NkwyMDQuMTEgNTY2LjU2TDE4OC4yIDYzMS43OEwxMzYuMDUgNjMxLjc4TDIwNy4yOSAzNjcuODhaTTM3Ny4wMSA0MTMuODRMMzc3LjAxIDM2Ny44OEw1MzQuMTQgMzY3Ljg4TDUzNC4xNCA0MTMuODRMNDgxLjY1IDQxMy44NEw0ODEuNjUgNTg1LjgzTDUzNC4xNCA1ODUuODNMNTM0LjE0IDYzMS43OEwzNzcuMDEgNjMxLjc4TDM3Ny4wMSA1ODUuODNMNDI5LjUgNTg1LjgzTDQyOS41IDQxMy44NEwzNzcuMDEgNDEzLjg0Wk01NjguODggNDA2LjA1TDYyNy43NCA0MDYuMDVMNjI3Ljc0IDQ3MC41N0w1NjguODggNDcwLjU3TDU2OC44OCA0MDYuMDVaTTU2OC44OCA1MjkuMDhMNjI3Ljc0IDUyOS4wOEw2MjcuNzQgNTkzLjk1TDU2OC44OCA1OTMuOTVMNTY4Ljg4IDUyOS4wOFpNODUzLjkgNjMxLjc4TDc5OS45OSA2MzEuNzhMNzQ5LjYyIDU0NC40Nkw2OTkuNDIgNjMxLjc4TDY0NS41MSA2MzEuNzhMNzIyLjkzIDQ5Ny44TDY0Ny42MyAzNjcuODhMNzAxLjU0IDM2Ny44OEw3NDkuNjIgNDUxLjg0TDc5Ny44NyAzNjcuODhMODUxLjc4IDM2Ny44OEw3NzYuODQgNDk3LjhMODUzLjkgNjMxLjc4WiIgLz4KPC9zdmc%2BCg%3D%3D&labelColor=13171f)](https://0rwx.com)

Exposes the state of a [Komodo](https://github.com/moghtech/komodo) Core as
Prometheus metrics: servers, stacks, containers, resource syncs, procedures and
repos. It is a thin wrapper around Komodo's read API, as suggested in
[moghtech/komodo#1600](https://github.com/moghtech/komodo/issues/1600), so no
agent runs next to the containers. It also serves the Komodo servers as
Prometheus HTTP service discovery, for scraping an exporter on every server.

Built against Komodo v2.3.

## Image

```sh
docker pull ghcr.io/niekkeijzer/komodo-exporter:latest
```

Published tags: `<major>` (from 1.0.0 on), `<major.minor>`,
`<major.minor.patch>` and `latest` for releases (`v*` git tags); `main` and
`sha-<short>` for the main branch.

## Configure

| Variable | Meaning |
|---|---|
| `KOMODO_URL` | Komodo Core, such as `http://komodo:9120` |
| `KOMODO_API_KEY`, `KOMODO_API_SECRET` | an API key with read access |
| `KOMODO_EXPORTER_SD_DOMAIN` | optional: domain appended to server names in `/sd/servers` |

Give the exporter a service user with read-only access. In a resource sync:

```toml
[[user_group]]
name = "monitoring"
users = ["komodo-exporter"]
all.Server = "Read"
all.Stack = "Read"
all.Repo = "Read"
all.ResourceSync = "Read"
all.Procedure = "Read"
```

Create the service user and its API key in the Komodo UI (Settings, Users).

## Run

```sh
export KOMODO_URL=http://komodo:9120 KOMODO_API_KEY="..." KOMODO_API_SECRET="..."
./komodo-exporter -listen :9821
```

`-version` prints the version and exits.

Every scrape of `/metrics` queries Komodo, with a 10 second timeout. Scrape it
at Komodo's own polling rate or slower.

## Metrics

States are exported one-hot, one series per state with value 1 for the
current one, like `kube_pod_status_phase`:

```promql
komodo_stack_state{state!~"running|deploying"} == 1
```

| Metric | Labels |
|---|---|
| `komodo_up`: whether the last query of the Komodo API succeeded; other metrics are left out when it didn't | |
| `komodo_scrape_duration_seconds` | |
| `komodo_exporter_build_info`: always 1 | `version`, `revision`, `goversion`, `goos`, `goarch`, … |
| `komodo_server_state`: `ok`, `not_ok` or `disabled` | `server`, `state` |
| `komodo_server_info`: the Periphery version | `server`, `version` |
| `komodo_server_cpu_usage_percent` | `server` |
| `komodo_server_memory_used_bytes`, `komodo_server_memory_total_bytes` | `server` |
| `komodo_server_disk_used_bytes`, `komodo_server_disk_total_bytes` | `server`, `mount` |
| `komodo_stack_state`: `running`, `unhealthy`, `down`, … | `stack`, `server`, `state` |
| `komodo_stack_project_missing` | `stack`, `server` |
| `komodo_container_state`: `running`, `exited`, `restarting`, … | `server`, `container`, `state` |
| `komodo_container_health_status`: `healthy`, `unhealthy`, `starting` or `none` | `server`, `container`, `status` |
| `komodo_container_cpu_usage_percent`, `komodo_container_memory_usage_bytes` | `server`, `container` |
| `komodo_resource_sync_state`: `ok`, `pending`, `failed`, … | `sync`, `state` |
| `komodo_procedure_state`: the last run, `ok`, `failed`, … | `procedure`, `state` |
| `komodo_repo_state`: `ok`, `failed`, … | `repo`, `state` |

Server stats come from Komodo's cache and are missing while a server is
disconnected.

## Service discovery

`/sd/servers` lists every enabled server as a target, labelled with its name.
With `KOMODO_EXPORTER_SD_DOMAIN=node.example.com`, server `periphery-01`
becomes target `periphery-01.node.example.com`. Add the port in the scrape
config, for example to scrape node-exporter on every server:

```yaml
scrape_configs:
  - job_name: node
    http_sd_configs:
      - url: http://komodo-exporter:9821/sd/servers
    relabel_configs:
      - source_labels: [__address__]
        target_label: __address__
        replacement: $1:9100
```

## Build and test

```sh
go build -o komodo-exporter ./cmd/komodo-exporter
go test -race ./...
```

## AI disclaimer

The code, documentation and CI workflow were drafted with AI assistance
(Claude) and reviewed before publishing.
