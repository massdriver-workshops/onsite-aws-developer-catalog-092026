---
templating: mustache
---

# timeoff-ui

{{#connections.namespace}}
Open `{{connections.namespace.ingress.scheme}}://{{connections.namespace.ingress.hostname}}{{connections.namespace.ingress.path_prefix}}/`.
{{/connections.namespace}}

## Image

Public image: `docker.io/massdrivercloud/hr-workshop-timeoff-ui`, tags on Docker Hub at https://hub.docker.com/r/massdrivercloud/hr-workshop-timeoff-ui. The `image.tag` parameter selects the release; every tag there is deployable.

## Page loads but shows "timeoff-api is not reachable"

The browser calls `{{connections.namespace.ingress.path_prefix}}{{params.timeoff_api_path}}` on the same host. Either no `timeoff-api` instance is deployed under that path, or its `path` param differs from `timeoff_api_path` here. Check the API directly:

```sh
curl -s {{connections.namespace.ingress.scheme}}://{{connections.namespace.ingress.hostname}}{{connections.namespace.ingress.path_prefix}}{{params.timeoff_api_path}}/info
```

## Payroll panel says "not connected"

Expected until a `payroll-api` instance is deployed at `{{params.payroll_api_path}}` under the same prefix. The page polls every five seconds and fills the panel in on its own once the API answers.

## 404 on the page itself

The entrypoint copies the site under `BASE_PATH` at start. If the ingress prefix changed after the pod started, restart:

```sh
kubectl -n {{connections.namespace.name}} rollout restart deploy/{{slug}}
kubectl -n {{connections.namespace.name}} logs deploy/{{slug}} | grep 'timeoff-ui serving'
```

## Stale page after a release

The static files are served with default caching. A hard refresh clears it. `config.json` is never cached, so API paths update on a normal reload.
