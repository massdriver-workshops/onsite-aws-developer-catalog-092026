---
templating: mustache
---

# timeoff-api

{{#resources.api}}
Answers at `{{resources.api.url}}`. `GET {{resources.api.url}}info` returns the running version and policy.
{{/resources.api}}

## Image

Public image: `docker.io/massdrivercloud/hr-workshop-timeoff-api`, tags on Docker Hub at https://hub.docker.com/r/massdrivercloud/hr-workshop-timeoff-api. The `image.tag` parameter selects the release; every tag there is deployable.

## Deploy waits, then fails after five minutes

The deploy waits for the application to report healthy, and health means the database answered. Open the Deployments tab and read the last deployment's log: the container's own log lines are near the end.

- `connect to database` means the database credential did not work. Check that the `mariadb` port is connected to a `mariadb` instance that deployed successfully.
- `DATABASE_URL is required` or `SESSION_SECRET is required` means a dependency or secret is missing. The form shows which.
- No application log lines at all means the image could not start. Check `image.repository` and `image.tag` against the tags on Docker Hub.

## Requests are accepted but payroll never sees them

Decisions are published to `{{params.events_topic}}`. Publish failures are logged, not returned to the caller, so the page looks fine while events go missing.

{{#resources.api}}
```sh
curl -s {{resources.api.url}}info
```
{{/resources.api}}

`events_enabled: false` means the `kafka` dependency did not resolve. `events_enabled: true` with nothing arriving at payroll means the two bundles picked different topics; `events_topic` is chosen independently on each. The Deployments tab log shows `events: publish ... for request N` with the reason when a publish fails.

## Requests rejected

`a single request may cover at most N days` is the `max_days_per_request` parameter doing its job. Raise it and redeploy.

`choose who you are acting as first` means the page sent no session token. Pick a name in the top right of the page.

## Rotate SESSION_SECRET

Set a new value on the instance's secrets and redeploy. Existing acting-as tokens stop validating immediately; the page fetches a new one when the user picks a name again. Nothing is stored in the database, so there is no migration.

## Roll back a release

Redeploy with the previous image tag. Schema changes are additive, so an older version runs against a newer schema.
