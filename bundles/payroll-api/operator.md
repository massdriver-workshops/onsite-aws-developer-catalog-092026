---
templating: mustache
---

# payroll-api

{{#connections.namespace}}
Reachable at `{{connections.namespace.ingress.scheme}}://{{connections.namespace.ingress.hostname}}{{connections.namespace.ingress.path_prefix}}{{params.path}}/`. `GET .../info` shows the version, the consumer group, and how many events have been consumed.
{{/connections.namespace}}

## Image

Public image: `docker.io/massdrivercloud/hr-workshop-payroll-api`, tags on Docker Hub at https://hub.docker.com/r/massdrivercloud/hr-workshop-payroll-api. The `image.tag` parameter selects the release; every tag there is deployable.

## Ledger is empty after approvals

Every start replays `{{params.events_topic}}` from the beginning, so an empty ledger means events are not arriving, not that state was lost.

```sh
kubectl -n {{connections.namespace.name}} logs deploy/{{slug}} | grep -i kafka
```

`consuming <topic> as group <group>` followed by nothing means the topic is empty or `timeoff-api` publishes to a different topic. Both bundles choose `events_topic` independently; check they match. A SASL or authorization error means the `kafka` resource's ACLs do not cover this topic or consumer group.

## Ledger shows the wrong numbers

The ledger is derived, so recomputing it is a restart:

```sh
kubectl -n {{connections.namespace.name}} rollout restart deploy/{{slug}}
```

Numbers that are still wrong after a replay reflect the events themselves. Look at `GET .../events` for the decisions that fed them.

## Pod restarts in a loop

The process exits at startup if `PAYMENTS_API_KEY` is missing or if any Kafka setting is absent. The first log line names the variable.

```sh
kubectl -n {{connections.namespace.name}} logs deploy/{{slug}} --previous --tail=5
```

## Rotate PAYMENTS_API_KEY

Set the new value on the instance's secrets and redeploy. The app reads it once at start; `GET .../info` shows the last four characters so you can confirm the new key is live.
