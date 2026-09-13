---
templating: mustache
---

# payroll-api

{{#resources.api}}
Answers at `{{resources.api.url}}`. `GET {{resources.api.url}}info` shows the version, the consumer group, and how many events have been consumed.
{{/resources.api}}

## Image

Public image: `docker.io/massdrivercloud/hr-workshop-payroll-api`, tags on Docker Hub at https://hub.docker.com/r/massdrivercloud/hr-workshop-payroll-api. The `image.tag` parameter selects the release; every tag there is deployable.

## Ledger is empty after approvals

Every start replays `{{params.events_topic}}` from the beginning, so an empty ledger means events are not arriving, not that state was lost.

{{#resources.api}}
```sh
curl -s {{resources.api.url}}info
```
{{/resources.api}}

`events_consumed: 0` after approvals means `timeoff-api` publishes to a different topic; both bundles choose `events_topic` independently and must agree. The Deployments tab log shows `consuming <topic> as group <group>` on start, and any authorization error after it.

## Ledger shows the wrong numbers

The ledger is derived from the events, so recomputing it is a redeploy: the process restarts and replays the topic. Numbers that are still wrong after a replay reflect the events themselves. `GET {{resources.api.url}}events` lists the decisions that fed them.

## Deploy fails immediately

The process exits at startup if `PAYMENTS_API_KEY` is missing or if the `kafka` dependency is not set. The first log line in the Deployments tab names what is missing.

## Rotate PAYMENTS_API_KEY

Set the new value on the instance's secrets and redeploy. The app reads it once at start; `GET {{resources.api.url}}info` shows the last four characters so you can confirm the new key is live.
