<div align="center">

<img src="https://cdn.simpleicons.org/ansible/EE0000" width="60" alt="Ansible" />
&nbsp;&nbsp;&nbsp;
<img src="https://cdn.simpleicons.org/grafana/F46800" width="60" alt="Grafana" />
&nbsp;&nbsp;&nbsp;
<img src="https://cdn.simpleicons.org/prometheus/E6522C" width="60" alt="Prometheus" />
&nbsp;&nbsp;&nbsp;
<img src="https://grafana.com/static/img/logos/logo-loki.svg" width="60" alt="Loki" />
&nbsp;&nbsp;&nbsp;
<img src="https://cdn.simpleicons.org/opentelemetry/425CC7" width="60" alt="OpenTelemetry" />

# observability-lab

**A fully automated observability stack on Azure — Terraform, Ansible, k3s, Prometheus, Grafana, Loki & Tempo with distributed tracing and SLO-based alerting**

![Terraform](https://img.shields.io/badge/IaC-Terraform-844FBA?style=flat-square&logo=terraform&logoColor=white)
![Ansible](https://img.shields.io/badge/Config-Ansible-EE0000?style=flat-square&logo=ansible&logoColor=white)
![Kubernetes](https://img.shields.io/badge/Kubernetes-k3s-326CE5?style=flat-square&logo=kubernetes&logoColor=white)
![Helm](https://img.shields.io/badge/Helm-v3-0F1689?style=flat-square&logo=helm&logoColor=white)
![Prometheus](https://img.shields.io/badge/Metrics-Prometheus-E6522C?style=flat-square&logo=prometheus&logoColor=white)
![Grafana](https://img.shields.io/badge/Dashboards-Grafana-F46800?style=flat-square&logo=grafana&logoColor=white)
![Loki](https://img.shields.io/badge/Logs-Loki-F46800?style=flat-square)
![Tempo](https://img.shields.io/badge/Traces-Tempo-F46800?style=flat-square)
![OpenTelemetry](https://img.shields.io/badge/Instrumentation-OpenTelemetry-425CC7?style=flat-square&logo=opentelemetry&logoColor=white)
![Go](https://img.shields.io/badge/Go-App-00ADD8?style=flat-square&logo=go&logoColor=white)
![Azure](https://img.shields.io/badge/Cloud-Azure-0078D4?style=flat-square&logo=microsoftazure&logoColor=white)

</div>

---

## What is this?

A self-built, end-to-end observability lab implementing the **three pillars of observability** — metrics, logs, and distributed traces — provisioned entirely as code on Azure.

**Terraform** stands up the infrastructure (VM, networking, a static public IP, and a daily auto-shutdown schedule to protect a fixed credit budget). **Ansible** then takes over completely: installing k3s, deploying the full monitoring stack via Helm (`kube-prometheus-stack`, `loki-stack`, and `tempo`), shipping a small Go service instrumented with **OpenTelemetry**, wiring up Traefik Ingress, and configuring SLO-based email alerting — all idempotently, so the whole stack can be rebuilt from scratch with two commands.

The Go app exports Prometheus metrics, structured JSON logs (with `trace_id` embedded), and OTLP traces — enabling **cross-signal correlation**: jump from a slow span in Tempo directly to the matching log line in Loki, or from a Prometheus exemplar directly to the trace.

---

## Architecture

```
Terraform (Azure, Japan East)
  └── VM (Standard_D2s_v3) + VNet/NSG + static Public IP + daily auto-shutdown
        └── Ansible (idempotent, role-based)
              ├── k3s            — single-node Kubernetes (Traefik ingress built in)
              ├── Helm
              │     ├── kube-prometheus-stack → Prometheus + Grafana + Alertmanager
              │     ├── loki-stack            → Loki + Promtail
              │     └── tempo                 → Distributed tracing backend
              ├── app            — Go service (OTEL-instrumented: metrics + logs + traces)
              └── ingress        — Traefik routes for Grafana & Prometheus

Go app (OpenTelemetry SDK)
  ├── /metrics  → Prometheus (scraped via ServiceMonitor) — with Exemplars
  ├── stdout    → Promtail → Loki (structured JSON, trace_id embedded)
  └── OTLP/gRPC → Tempo (spans per request)

Grafana
  ├── Prometheus datasource  → metrics dashboards + exemplar → trace links
  ├── Loki datasource        → log explorer
  └── Tempo datasource       → trace explorer + trace → log correlation
```

---

## Stack

| Layer | Tool | Role |
|---|---|---|
| IaC | **Terraform** | Provisions Azure VM, VNet/NSG, static public IP, auto-shutdown |
| Config management | **Ansible** | Installs k3s, deploys every Helm chart, the app, and Ingress — fully idempotent |
| Orchestration | **k3s** | Lightweight single-node Kubernetes cluster |
| Packaging | **Helm v3** | Deploys `kube-prometheus-stack`, `loki-stack`, and `tempo` |
| Metrics & Alerts | **Prometheus + Alertmanager** | Scrapes `/metrics` with exemplars, evaluates SLO rules, sends alerts |
| Dashboards | **Grafana** | SLI dashboard, cluster & node dashboards, Loki log explorer, Tempo trace explorer |
| Logs | **Loki + Promtail** | Aggregates structured JSON logs with embedded `trace_id` from every pod |
| Traces | **Tempo** | Stores and queries distributed traces from the Go app |
| Instrumentation | **OpenTelemetry SDK** | Auto-instruments the Go/Gin app — exports spans via OTLP/gRPC to Tempo |
| Correlation | **Exemplars + Derived fields** | Metrics → Trace, Trace → Logs cross-signal navigation in Grafana |
| Ingress | **Traefik** | Built into k3s — exposes Grafana & Prometheus externally |
| App | **Go + Gin** | Instrumented API simulating realistic traffic, latency variance, and failures |
| Notifications | **Alertmanager → Gmail SMTP** | Email on SLO breach and on resolve |

---

## SLI / SLO / SLA

Full definitions and PromQL queries live in [`sli-slo/definitions.md`](sli-slo/definitions.md).

### SLI
- **Availability** — percentage of `/work` requests that do *not* return a 5xx status
- **Latency (p95)** — 95th percentile response time for `/work`

### SLO

| SLI | Target |
|---|---|
| Availability | ≥ 90% |
| p95 Latency | < 200ms |

> The target is intentionally 90% rather than a typical 99.5% — the app simulates a ~10% failure rate on purpose, to generate realistic signal for this lab.

### SLA
85% monthly availability for `/work`, measured over rolling 30-day windows — kept below the internal SLO to leave error-budget margin for planned maintenance.

---

## Repository Structure

```
observability-lab/
├── ansible/
│   ├── group_vars/          ← secrets.yml (gitignored): SMTP + Grafana admin password
│   ├── roles/               ← helm, prometheus, loki, tempo, ingress, app
│   ├── .gitignore
│   ├── inventory.ini
│   └── playbook.yaml
├── app/
│   ├── .dockerignore
│   ├── Dockerfile
│   ├── go.mod
│   ├── go.sum
│   └── main.go              ← OTEL SDK + Prometheus metrics + structured logs
├── docs/
│   ├── screenshots/
│   └── grafana-dashboard.json
├── sli-slo/
│   └── definitions.md
└── terraform/
    ├── main.tf
    ├── outputs.tf
    ├── providers.tf
    └── variables.tf
```

---

## Screenshots

### Full `ansible-playbook` run, applied idempotently end to end
![Ansible run](docs/screenshots/ansible.png)

### Every component running across `app`, `monitoring`, `logging`, `tracing`, and `kube-system`
![Pods](docs/screenshots/pods.png)

### Live SLI dashboard — availability, p95 latency, and request rate for `/work`
![SLI Dashboard](docs/screenshots/grafana-sli.png)

### Kubernetes cluster compute resources
![Cluster dashboard](docs/screenshots/grafana-cluster.png)

### Node Exporter system metrics
![Node dashboard](docs/screenshots/grafana-node.png)

### Structured application logs queried live through Grafana Explore → Loki
![Loki Explore](docs/screenshots/grafana-logs.png)

### Distributed traces from the Go app — Tempo trace list with service and operation
![Tempo traces](docs/screenshots/tempo-traces.png)

### Single trace waterfall — span duration and attributes for a `GET /work` request
![Tempo span](docs/screenshots/tempo-span.png)

### Trace-to-Logs correlation — jumping from a Tempo span directly to the matching Loki log line
![Trace to logs](docs/screenshots/tempo-trace-to-logs.png)

### SLOAvailabilityBreach moving into PENDING as the error budget runs out
![Prometheus alert pending](docs/screenshots/prometheus-alerts.png)

### The same alert FIRING once the breach holds for the configured 2 minutes
![Prometheus alert firing](docs/screenshots/prometheus-firing.png)

### Target health — app, Grafana, and Alertmanager all scraping successfully
![Prometheus targets](docs/screenshots/prometheus-targets.png)

### Email notification sent the moment the SLO breach fires
![Alert email — firing](docs/screenshots/Screenshot_20260626_131943_Gmail.jpg)

### Resolved notification once availability recovered
![Alert email — resolved](docs/screenshots/Screenshot_20260626_132801_Gmail.jpg)

---

## App Endpoints

| Endpoint | Description |
|---|---|
| `GET /health` | Liveness check |
| `GET /ready` | Readiness check |
| `GET /work` | Simulated workload, ~10% failure rate — the endpoint behind every SLI/SLO |
| `GET /metrics` | Prometheus scrape endpoint (includes exemplars linking to Tempo) |

---

## Alerting

Two Prometheus rules evaluate the SLOs continuously:

- **`SLOAvailabilityBreach`** — fires when `/work` availability drops below 90% for 2+ minutes
- **`SLOLatencyBreach`** — fires when p95 latency exceeds 200ms for 2+ minutes

Alertmanager is configured with Gmail SMTP and sends an email on both `firing` and `resolved` transitions.

---

<div align="center">
<sub>Part of a DevOps portfolio — <a href="https://github.com/amirhosssein0/terraform-lab">terraform-lab</a> | <a href="https://github.com/amirhosssein0/vault-cicd-lab">vault-cicd-lab</a></sub>
</div>