# Prism

A distributed A/B testing and feature flag platform — deterministic assignment, streaming metric ingestion, statistical significance testing, and live experiment dashboards.

I had worked with data pipelines and analytics before, but kept reading engineering posts from Netflix, Meta, and LinkedIn about how they test product changes before shipping them broadly. I wanted to understand how that infrastructure actually works, so I built it — the assignment logic, the event pipeline, and the stats engine from scratch.

## What's Inside

**Deterministic assignment** — users are assigned to variants by hashing `experimentID:userID` with SHA-256 and mapping onto cumulative weight buckets. The same user always gets the same variant across server restarts, with no storage required. Supports any traffic split: 50/50, 90/10, 1/99, or k-arm.

**Streaming metric pipeline** — metric events are published to Redpanda (Kafka-compatible) and consumed by a Go consumer that writes to Postgres with idempotency guarantees. Duplicate Kafka message deliveries are silently ignored via `ON CONFLICT DO NOTHING`.

**Statistical significance engine** — a Python gRPC service computes two-proportion z-tests with Wilson score confidence intervals. Bonferroni correction is applied for multi-arm experiments. The false positive rate is validated at ~5% via Monte Carlo simulation.

**Feature flags** — percentage rollout using the same hash-based approach as experiment assignment, with user-level overrides and a global kill switch.

## Architecture

```
┌─ prism-api (Go) ──────────────────────────────────────┐
│  POST /assign    → SHA-256 hash → variant              │
│  POST /events    → Redpanda → prism-consumer           │
│  GET  /results   → prism-stats (gRPC) → p-value        │
│  GET  /flags     → hash-based rollout evaluation       │
└───────────────────────────────────────────────────────┘
         │                              │
         ▼                              ▼
  prism-consumer (Go)          prism-stats (Python)
  Redpanda → Postgres          z-test, Wilson CIs
  idempotent writes            Bonferroni correction
```

## Quick Start

Requires Go, Python 3.11+, and Docker Desktop.

```bash
cp .env.example .env        # fill in your values
docker compose up -d
docker exec prism-redpanda-1 rpk topic create prism.metric.events
bash scripts/migrate.sh
```

Start each service in a separate terminal:

```bash
go run ./cmd/api
go run ./cmd/consumer
cd stats && source .venv/bin/activate && python server.py
```

Run the demo — 4,850 simulated users, treatment converges to significance in ~3 minutes:

```bash
python scripts/traffic_gen.py --rate 30 --duration 180
open http://localhost:3000   # Grafana dashboard
```

## Benchmarks

Measured on Apple M4 Pro:

| Benchmark              | ns/op | ops/sec |
|------------------------|-------|---------|
| Single-core assignment | 199   | ~5M     |
| Parallel assignment    | 95    | ~10M    |

## Status

- [x] Deterministic weighted variant assignment
- [x] Experiment API (create, read, list, update status)
- [x] Docker Compose infrastructure (Postgres, Redis, Redpanda, Prometheus, Grafana)
- [x] Kafka metric event pipeline with idempotent consumer
- [x] Statistical significance engine (Python gRPC, z-test, Wilson CIs)
- [x] Prometheus metrics instrumentation
- [x] Feature flags (percentage rollout, user overrides)
- [x] Grafana dashboard (conversion rates, p-value, ingestion rate)
- [x] Traffic generator for live demo

## Inspiration

Netflix Experimentation Platform, Meta Gatekeeper, LinkedIn XLNT