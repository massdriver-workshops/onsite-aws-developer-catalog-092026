# timeoff-ui

The browser front end for the time-off stack: file a request, approve or deny it, and watch the payroll impact land next to it. Static files served by nginx; all data comes from `timeoff-api` and `payroll-api` over the same ingress.

## Dependencies

| Dependency | Resource type | Used for |
|-----------|---------------|----------|
| `landing_zone` | `landing-zone` | Where the pods run and the hostname the page is served on |
| `timeoff_api` | `api-endpoint` | Where the time-off API answers |
| `payroll_api` | `api-endpoint`, optional | Where the payroll API answers; the panel waits until connected |

No database, no Kafka, no secrets. The UI reads the API URLs from its connections.

## Emits

`site`: an `api-endpoint` whose `url` is the page. Open it from the Resources tab.
