# Benchmark Results

Machine: Apple M4 Pro, darwin/arm64
Date: 2025-05-18

## Assignment (pkg/assignment)

| Benchmark | ns/op | B/op | allocs/op |
|---|---|---|---|
| BenchmarkAssign | 199.8 | 232 | 9 |
| BenchmarkAssignSameUser | 161.5 | 272 | 8 |
| BenchmarkAssignParallel | 95.18 | 232 | 9 |

~5M assignments/sec single-core, ~10M/sec parallel.
