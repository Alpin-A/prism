# Prism

A distributed A/B testing and feature flag platform with deterministic assignment, streaming metric ingestion, statistical significance testing, and live experiment dashboards.

I had worked with data pipelines and analytics before, but kept reading engineering posts from Netflix, Meta, and LinkedIn about how they test product changes before shipping them broadly. I wanted to understand how that infrastructure actually works. Not just call an SDK, but build the assignment logic, the event pipeline, and the stats engine from scratch. Prism is that project.

## Status

Under active development. Currently implemented:
- [x] Deterministic weighted variant assignment
- [x] Experiment API (create, read, list, update status)
- [x] Docker Compose infrastructure (Postgres, Redis, Redpanda, Prometheus, Grafana)
- [x] Kafka metric event pipeline with idempotent consumer
- [x] Statistical significance engine (Python gRPC, z-test, Wilson CIs)
- [x] Prometheus metrics instrumentation
- [x] Feature flags (percentage rollout, user overrides)
- [ ] Grafana dashboard
- [ ] Traffic generator / demo

## How Assignment Works

A user is assigned to a variant by hashing `experimentID:userID` with SHA-256 and mapping the result onto cumulative weight buckets. Including the experiment ID in the hash means the same user gets independent assignments across different experiments. No storage is required to compute an assignment.

Supports any traffic split: 50/50, 90/10, 1/99, or k-arm.

## Benchmarks

Measured on Apple M4 Pro:

| Benchmark | ns/op | ops/sec |
|---|---|---|
| Single-core | 199 | ~5M |
| Parallel | 95 | ~10M |

## Inspiration

Netflix Gibbs, Meta Gatekeeper, LinkedIn XLNTs