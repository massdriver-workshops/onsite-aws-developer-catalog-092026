# timeoff-ui

The browser front end for the time-off stack: file a request, approve or deny it, and watch the payroll impact land next to it. Static files served by nginx; all data comes from `timeoff-api` and `payroll-api` over the same ingress.

## Dependencies

| Dependency | Resource type | Used for |
|-----------|---------------|----------|
| `namespace` | `namespace` | Where the pods run and the path prefix the page is served under |

No database, no Kafka, no secrets. The UI only needs to know where the two APIs live, and both default to the paths their bundles use.

## Parameters worth knowing

- `timeoff_api_path` and `payroll_api_path` are joined to the namespace's prefix at deploy time.
- The page works with only `timeoff-api` deployed. The payroll panel fills in when `payroll-api` appears.
