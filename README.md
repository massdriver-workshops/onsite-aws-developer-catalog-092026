# Getting started

You have your own organization on Massdriver. In it you will find a project, an environment, and one resource already set as the environment default: your **landing zone**. It holds a MariaDB credential, a Kafka credential, and a hostname that is yours. Everything you deploy connects to it. You never provision infrastructure; you build applications.

## 1. Sign in

Open the invitation email, sign in with GitHub, and pick your organization. The GitHub account must use the same email address the invitation went to.

## 2. Look at the canvas

Open the project and its environment. The landing zone is listed under environment defaults. Click it. You will see your hostname, the databases and topics made for you, and the credentials, masked.

## 3. Add your data connections

Drag `mariadb` from the bundle list onto the canvas. Pick the `timeoff` database. Deploy. It takes a few seconds. Open its Resources tab: a connection your applications can use, with the password shown as `[SENSITIVE]`.

Drag `kafka` onto the canvas. Deploy. Same thing for your topics.

These two bundles create nothing. They turn what is already yours into typed connections.

## 4. Deploy the time-off app

Drag `timeoff-api`. Connect its `mariadb` and `kafka` ports to the two resources you just made. In the form, pick the events topic from the dropdown. Under Secrets, set `SESSION_SECRET` to any random string; deploy is blocked until you do. Deploy. About a minute.

Drag `timeoff-ui`. Connect its `timeoff_api` port to `timeoff-api`. Deploy. Open the URL from its outputs. File a time-off request, then approve it.

## 5. Add payroll

Drag `payroll-api`. Connect its `kafka` port. Pick the same events topic. Set `PAYMENTS_API_KEY` under Secrets to anything. Deploy. Then connect `timeoff-ui`'s optional `payroll_api` port to it and redeploy the UI. Reload the page: the payroll panel now shows the ledger, built from the events the API published.

## What is in this catalog

| Bundle | Connects to | What it does | Image |
|--------|-------------|--------------|-------|
| `mariadb` | landing zone | One database from your landing zone as a `mariadb-authentication` connection | |
| `kafka` | landing zone | Your topics as a `kafka-authentication` connection | |
| `timeoff-api` | `mariadb`, `kafka` | Requests and approvals. Publishes a decision event per approval or denial. Emits its endpoint. | [coryodaniel/hr-workshop-timeoff-api](https://hub.docker.com/r/coryodaniel/hr-workshop-timeoff-api) |
| `payroll-api` | `kafka` | Consumes decision events into a PTO ledger. Emits its endpoint. | [coryodaniel/hr-workshop-payroll-api](https://hub.docker.com/r/coryodaniel/hr-workshop-payroll-api) |
| `timeoff-ui` | `timeoff-api`, `payroll-api` (optional) | The web page. Reads both API endpoints from its connections. | [coryodaniel/hr-workshop-timeoff-ui](https://hub.docker.com/r/coryodaniel/hr-workshop-timeoff-ui) |

Your applications answer at:

```
http://<your hostname>/              timeoff-ui
http://<your hostname>/api/timeoff/  timeoff-api
http://<your hostname>/api/payroll/  payroll-api
```

## Images

The example images are public on Docker Hub: [hr-workshop-timeoff-api](https://hub.docker.com/r/coryodaniel/hr-workshop-timeoff-api), [hr-workshop-payroll-api](https://hub.docker.com/r/coryodaniel/hr-workshop-payroll-api), [hr-workshop-timeoff-ui](https://hub.docker.com/r/coryodaniel/hr-workshop-timeoff-ui). Every bundle's `image` parameter defaults to them. Point it at your own image to run your own code.

## Build your own

Every bundle here is a folder with a `massdriver.yaml`, a `src/` directory of IaC, an `operator.md` runbook, and an icon. Copy one, change the image and the parameters, and publish it:

```sh
cd bundles/timeoff-api
mass bundle build
mass bundle lint
mass bundle publish --development
```

Your bundle appears in your organization's bundle list and works on the canvas like the others. The example application code is under `example/apps`, with a `docker-compose.yml` that runs the whole stack on a laptop.

## Things to know

- A required connection disables the deploy button until it is connected. That is not an error.
- A required secret does the same.
- Your database credential reaches your databases and nothing else. Your Kafka credential reaches your topics and nothing else. Both are enforced by the servers.
- Each landing zone has limits on database connections, Kafka throughput, and compute. Hitting one gives your application an error; it does not affect anyone else.
