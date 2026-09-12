# timeoff-ui

{{#dependencies.namespace}}
Open `{{dependencies.namespace.ingress.scheme}}://{{dependencies.namespace.ingress.hostname}}{{dependencies.namespace.ingress.path_prefix}}/`.
{{/dependencies.namespace}}

## Page loads but shows "timeoff-api is not reachable"

The browser calls `{{dependencies.namespace.ingress.path_prefix}}{{params.timeoff_api_path}}` on the same host. Either no `timeoff-api` instance is deployed under that path, or its `path` param differs from `timeoff_api_path` here. Check the API directly:

```sh
curl -s {{dependencies.namespace.ingress.scheme}}://{{dependencies.namespace.ingress.hostname}}{{dependencies.namespace.ingress.path_prefix}}{{params.timeoff_api_path}}/info
```

## Payroll panel says "not connected"

Expected until a `payroll-api` instance is deployed at `{{params.payroll_api_path}}` under the same prefix. The page polls every five seconds and fills the panel in on its own once the API answers.

## 404 on the page itself

The entrypoint copies the site under `BASE_PATH` at start. If the ingress prefix changed after the pod started, restart:

```sh
kubectl -n {{dependencies.namespace.name}} rollout restart deploy/{{id}}
kubectl -n {{dependencies.namespace.name}} logs deploy/{{id}} | grep 'timeoff-ui serving'
```

## Stale page after a release

The static files are served with default caching. A hard refresh clears it. `config.json` is never cached, so API paths update on a normal reload.
