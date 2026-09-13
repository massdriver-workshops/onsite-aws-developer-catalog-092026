# Time off

Three services that make up a small time-off product: employees file requests, a manager approves or denies them, and payroll sees the result as events.

| Service | What it does | Depends on |
|---------|--------------|------------|
| `services/timeoff-api` | Go API. Requests, approvals, denials. Owns a MariaDB schema. Publishes `timeoff.requested` and `timeoff.decided` events. | MariaDB, Kafka |
| `services/payroll-api` | Go API. Consumes decision events into an in-memory PTO ledger with pay-period adjustments. No database, no writes. | Kafka |
| `services/timeoff-ui` | Static page served by nginx. Calls both APIs by path on the same host. | The two APIs |

Each service has a bundle under `bundles/` that deploys it on the attendee's hostname. The bundles take a `landing-zone` dependency and, where needed, `mariadb-authentication` and `kafka-authentication`. They never take a cloud credential.

## Run it locally

```sh
docker compose up --build
open http://localhost:8000/local/
```

Compose runs MariaDB, a Kafka-compatible broker, the three services, and an nginx front door that routes by path the same way the cluster ingress does. Pick a name in the top right, file a request, approve it, and watch the payroll panel update.

## Configuration

Every service reads its settings from the environment. `BASE_PATH` is the prefix the service is reachable under; nothing rewrites paths in front of it.

### timeoff-api

| Variable | Required | Meaning |
|----------|----------|---------|
| `DATABASE_URL` | yes | `mysql://user:pass@host:3306/schema`, optionally `?tls=preferred` |
| `SESSION_SECRET` | yes | Signs acting-as tokens |
| `KAFKA_BROKERS`, `KAFKA_TOPIC` | no | Events are disabled when either is missing |
| `KAFKA_USERNAME`, `KAFKA_PASSWORD`, `KAFKA_SASL_MECHANISM`, `KAFKA_TLS` | no | SCRAM auth; TLS defaults on when a username is set |
| `MAX_DAYS_PER_REQUEST` | no | Default 30 |
| `AUTO_APPROVE`, `APPROVAL_WINDOW_DAYS` | no | Default false and 5 |
| `NAMESPACE`, `COMPANY_NAME`, `BASE_PATH`, `PORT` | no | Display and routing |

### payroll-api

| Variable | Required | Meaning |
|----------|----------|---------|
| `PAYMENTS_API_KEY` | yes | Checked for presence, shown masked |
| `KAFKA_BROKERS`, `KAFKA_TOPIC`, `KAFKA_CONSUMER_GROUP` | yes | The topic is replayed from the start on every boot |
| `KAFKA_USERNAME`, `KAFKA_PASSWORD`, `KAFKA_SASL_MECHANISM`, `KAFKA_TLS` | no | SCRAM auth |
| `PAY_PERIOD` | no | `weekly`, `biweekly`, `semimonthly`, `monthly`; default `biweekly` |
| `DEMO_DAILY_RATE_CENTS` | no | Default 32000 |
| `NAMESPACE`, `BASE_PATH`, `PORT` | no | Display and routing |

### timeoff-ui

| Variable | Required | Meaning |
|----------|----------|---------|
| `BASE_PATH` | no | Where the page is served |
| `TIMEOFF_API_URL`, `PAYROLL_API_URL` | no | Default to `$BASE_PATH/api/timeoff` and `$BASE_PATH/api/payroll` |
| `NAMESPACE` | no | Shown in the header |

## Event contract

`timeoff-api` publishes JSON to one topic, keyed by employee name. `payroll-api` acts on `timeoff.decided`.

```json
{
  "type": "timeoff.decided",
  "version": 1,
  "request_id": 12,
  "employee_id": 3,
  "employee": "Priya Natarajan",
  "team": "Customer Success",
  "kind": "vacation",
  "start_date": "2026-10-05",
  "end_date": "2026-10-09",
  "days": 5,
  "status": "approved",
  "decided_by": "Mei Tanaka",
  "namespace": "priya",
  "occurred_at": "2026-09-11T18:04:12Z"
}
```

## Images

Images are published to Docker Hub as `massdrivercloud/hr-workshop-<service>`. `.github/workflows/images.yml` builds all three on push to `main` and on tags.

## Bundles

Each bundle directory is publishable on its own:

```sh
cd bundles/timeoff-api
mass bundle build
mass bundle lint
mass bundle publish --development
```

The `landing-zone`, `mariadb-authentication`, `kafka-authentication`, and `api-endpoint` resource types must be published to the organization before `mass bundle build` can resolve them.
