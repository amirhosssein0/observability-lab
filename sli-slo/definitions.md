# SLI / SLO / SLA — Observability Lab

## SLI (Service Level Indicators)

### Availability
Percentage of `/work` requests that do NOT return a 5xx status code.

```promql
sum(rate(http_requests_total{path="/work", status!~"5.."}[5m]))
/
sum(rate(http_requests_total{path="/work"}[5m]))
```

### Latency (p95)
95th percentile response time for the `/work` endpoint.

```promql
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket{path="/work"}[5m]))
```

## SLO (Service Level Objectives)

| SLI | Target |
|---|---|
| Availability | >= 90% |
| p95 Latency | < 200ms |

> Note: the target is intentionally set to 90% (not 99.5%) because the
> application simulates a ~10% failure rate on purpose, to generate
> realistic signal for this lab.

## SLA (Service Level Agreement)

We commit to 85% monthly availability for the `/work` endpoint,
measured over rolling 30-day windows. This is set below the internal
SLO to leave error-budget margin for planned maintenance.