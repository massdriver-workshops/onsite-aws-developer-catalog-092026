# timeoff-api

{{#dependencies.namespace}}
Reachable at `{{dependencies.namespace.ingress.scheme}}://{{dependencies.namespace.ingress.hostname}}{{dependencies.namespace.ingress.path_prefix}}{{params.path}}/`. `GET .../info` returns the running version and policy.
{{/dependencies.namespace}}

## Image

Public image: `docker.io/coryodaniel/hr-workshop-timeoff-api`, tags on Docker Hub at https://hub.docker.com/r/coryodaniel/hr-workshop-timeoff-api. The `image.tag` parameter selects the release; every tag there is deployable.

## Pods never become ready

The readiness probe hits `/healthz`, which pings the database. A pod that stays unready cannot reach MariaDB.

```sh
kubectl -n {{dependencies.namespace.name}} get pods -l app.kubernetes.io/instance={{id}}
kubectl -n {{dependencies.namespace.name}} logs deploy/{{id}} --tail=50
```

{{#dependencies.mariadb}}
Look for `connect to database` in the logs. Confirm the schema user can log in from inside the cluster:

```sh
kubectl -n {{dependencies.namespace.name}} run -it --rm mysql-check --image=mariadb:11 --restart=Never -- \
  mariadb -h {{dependencies.mariadb.auth.hostname}} -P {{dependencies.mariadb.auth.port}} -u {{dependencies.mariadb.auth.username}} -p {{dependencies.mariadb.auth.database}} -e 'SELECT 1'
```

The user's grants stop at `{{dependencies.mariadb.auth.database}}`. If a migration fails with a permission error, the fix is in the landing zone, not here.
{{/dependencies.mariadb}}

## Requests are accepted but payroll never sees them

Decisions are published to `{{params.events_topic}}`. Publish failures are logged, not returned to the caller, so the UI looks fine while events go missing.

```sh
kubectl -n {{dependencies.namespace.name}} logs deploy/{{id}} | grep 'events:'
```

`events disabled` at startup means the Kafka dependency did not resolve. `publish ... for request N` with a SASL error means the credentials on the `kafka` resource are wrong or its ACLs do not cover the topic. Confirm `payroll-api` is consuming the same topic; the two bundles pick it independently.

## Rotate SESSION_SECRET

Set a new value on the instance's secrets and redeploy. Existing acting-as tokens stop validating immediately; the UI fetches a new one when the user picks a name again. Nothing is stored in the database, so there is no migration.

## Roll back a release

Redeploy with the previous image tag. Schema changes are additive, so an older version runs against a newer schema.
