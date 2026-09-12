# timeoff-api

Employees file time-off requests, managers approve or deny them, and every decision is published as an event for payroll to consume.

## Dependencies

| Dependency | Resource type | Used for |
|-----------|---------------|----------|
| `namespace` | `namespace` | Where the pods run and the path prefix they answer on |
| `mariadb` | `mariadb` | The schema this service owns; migrations run on start |
| `kafka` | `kafka` | Publishing `timeoff.requested` and `timeoff.decided` events |

## Secrets

`SESSION_SECRET` signs the acting-as tokens the UI sends. Deploy is blocked until it is set.

## Parameters worth knowing

- `events_topic` is a dropdown fed from the connected `kafka` resource. Pick the same topic in `payroll-api`.
- `path` cannot change after the first deploy; `timeoff-ui` is configured against it.
- `approval.window_days` only appears when `auto_approve` is off.
