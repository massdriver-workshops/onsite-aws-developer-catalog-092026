# timeoff-api

Employees file time-off requests, managers approve or deny them, and every decision is published as an event for payroll to consume.

## Dependencies

| Dependency | Resource type | Used for |
|-----------|---------------|----------|
| `landing_zone` | `landing-zone` | Where the pods run and the hostname they answer on |
| `mariadb` | `mariadb-authentication` | The schema this service owns; migrations run on start |
| `kafka` | `kafka-authentication` | Publishing `timeoff.requested` and `timeoff.decided` events |

## Emits

`api`: an `api-endpoint` with the URL this service answers on. `timeoff-ui` connects to it.

## Alarms

One alarm, **Replicas unavailable**, is registered on the instance and shows on its Alarms tab and in the dependency graph. It changes state when a monitor posts to the instance's alarm webhook; this workshop wires no monitor, so it stays quiet.

## Secrets

`SESSION_SECRET` signs the acting-as tokens the UI sends. Deploy is blocked until it is set.

## Parameters worth knowing

- `events_topic` is a dropdown fed from the connected `kafka` resource. Pick the same topic in `payroll-api`.
- `path` cannot change after the first deploy; `timeoff-ui` is configured against it.
- `approval.window_days` only appears when `auto_approve` is off.
