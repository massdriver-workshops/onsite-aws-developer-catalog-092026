# payroll-api

Consumes the decision events `timeoff-api` publishes and turns them into a PTO ledger with pay-period adjustments. It has no database and no write endpoints. Every start replays the topic, so the ledger is always derived from the events.

## Dependencies

| Dependency | Resource type | Used for |
|-----------|---------------|----------|
| `landing_zone` | `landing-zone` | Where the pod runs and the hostname it answers on |
| `kafka` | `kafka-authentication` | Consuming `timeoff.decided` events as the granted consumer group |

## Emits

`api`: an `api-endpoint` with the URL this service answers on. `timeoff-ui` connects to it.

## Alarms

One alarm, **Replicas unavailable**, is registered on the instance and shows on its Alarms tab and in the dependency graph. It changes state when a monitor posts to the instance's alarm webhook; this workshop wires no monitor, so it stays quiet.

## Secrets

`PAYMENTS_API_KEY` stands in for a real payments provider credential. Nothing on the canvas can supply it, which is the point: it is set on the instance, and deploy is blocked until it is. The app only checks it is present and shows its last four characters.

## Parameters worth knowing

- `events_topic` must match the topic chosen in `timeoff-api`.
- There is no replica count. One process consumes the topic so the ledger sees every event.
