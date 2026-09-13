---
templating: mustache
---

# timeoff-ui

{{#resources.site}}
Open `{{resources.site.url}}`.
{{/resources.site}}

## Image

Public image: `docker.io/massdrivercloud/hr-workshop-timeoff-ui`, tags on Docker Hub at https://hub.docker.com/r/massdrivercloud/hr-workshop-timeoff-ui. The `image.tag` parameter selects the release; every tag there is deployable.

## Page loads but says the API is not reachable

The page calls the URL on its `timeoff_api` dependency.

{{#dependencies.timeoff_api}}
```sh
curl -s {{dependencies.timeoff_api.url}}info
```
{{/dependencies.timeoff_api}}

No answer means the `timeoff-api` instance is still starting or its last deploy failed; check its Deployments tab. The page retries every five seconds on its own.

## Payroll panel says not connected

Expected until a `payroll-api` instance is connected to the optional `payroll_api` port and the UI is redeployed. The page polls every five seconds and fills the panel in when the API answers.

## Stale page after a release

Static files are served with default caching. A hard refresh clears it. The page's configuration is never cached, so API URLs update on a normal reload after a redeploy.
