# Massdriver developer workshop

This is everything you do in the 90-minute session, written so you can follow it on your own if you miss the room. Nothing here needs cloud credentials or infrastructure knowledge.

## What you have

You created your own organization on Massdriver. Everything below was put in it for you:

- a project called **Time Off** with an environment called **Dev**
- one resource already set as that environment's default: your **landing zone**
- the workshop bundles in your bundle list

The landing zone is your slice of the shared platform. It holds a MariaDB credential that reaches only your databases, a Kafka credential that reaches only your topics, and a hostname that is yours. Everything you deploy connects to it.

## Before the session

1. Sign in with GitHub at the URL in your workshop invitation. Use the account you registered with.
2. Open your organization. Open the **Time Off** project, then the **Dev** environment.
3. Click the landing zone under environment defaults. You see your hostname, two databases, two topics, and credentials shown as `[SENSITIVE]`.
4. Open the bundle list. You see `mariadb`, `kafka`, `timeoff-api`, `payroll-api`, and `timeoff-ui`.

If any of that is missing, say so in the workshop channel before the session starts. Do not fix it yourself.

## Part 1: the mental model (0:00)

Six words carry the whole product:

| Word | Meaning |
|------|---------|
| Bundle | A versioned package of infrastructure code plus the form that configures it. Single purpose: `mariadb`, `timeoff-api`. Generally one for each application, kept with your source code. For cloud services, one for each use case. |
| Resource type | A typed contract for what a bundle emits, such as `mariadb-authentication`. Ports on the canvas are typed by these. |
| Resource | A live copy of a resource type produced by a deployed bundle. Your landing zone is one. |
| Project | A group of related work, and the unit you replicate. Yours is Time Off. |
| Environment | A clone of a project: dev, staging, prod, a pull request. Each has a canvas. Yours is Dev. |
| Instance | A bundle placed and configured on a canvas. This is live infrastructure or a live application. |

The platform team owns infrastructure bundles. You own your application bundles and your canvases.

## Part 2: the lab (0:18)

You build the same stack the presenter just built. About fifteen minutes.

### Data resources

1. Drag **`mariadb`** from the bundle list onto the canvas. Its one port fills from the landing zone on its own. In the form, pick the `timeoff` database. Deploy. A few seconds.
2. Open its **Resources** tab. That is a `mariadb-authentication` resource: host, port, database, username, a connection URL, and the password shown as `[SENSITIVE]`. This credential reaches the databases in your landing zone and nothing else.
3. Drag **`kafka`**. Deploy. Its resource lists your topics.

Neither bundle creates anything. They turn what is already yours into typed resources an application can depend on. They exist to keep this workshop cheap: one shared database server and one shared Kafka cluster serve the whole room.

### The API

4. Drag **`timeoff-api`**. Connect its `mariadb` port to your `mariadb` instance and its `kafka` port to your `kafka` instance. Try connecting `kafka` to the `mariadb` port: the canvas refuses, because the types do not match.
5. In the form, pick the **events topic** from the dropdown. Those options come from your `kafka` resource. Look at **max days per request**: it has a minimum and a maximum with a readable message. Try 400.
6. Under **Secrets**, set `SESSION_SECRET` to any random string. The deploy button is disabled until you do. Deploy. About a minute while the container starts.

Ports carry first-party values: things Massdriver models and hands you over a connection. Secrets carry third-party values: anything Massdriver does not model, such as a Stripe key.

### The page

7. Drag **`timeoff-ui`**. Connect its `timeoff_api` port to `timeoff-api`. Leave the optional `payroll_api` port empty; optional ports never block a deploy. Deploy.
8. Open the **Resources** tab on `timeoff-ui` and click the site URL. Pick a name in the top right, file a time-off request, then approve it.

### If you have time

- Drag **`payroll-api`**. Connect its `kafka` port. Pick the same events topic. Set `PAYMENTS_API_KEY` under Secrets to anything. Deploy.
- Connect `timeoff-ui`'s optional `payroll_api` port to `payroll-api` and redeploy the UI. Reload the page: the payroll panel shows a ledger built from the events the API published. Approve another request and watch it update.
- Click any instance: **Config**, **Deployments**, **Resources**, **Alarms**. Deployments has every plan and apply, with logs.
- Open the Resources tab on `mariadb` and download the resource. That file is what an application receives.

### If something does not work

| What you see | Why | What to do |
|--------------|-----|------------|
| Deploy button disabled | A required port is not connected, or a required secret is not set | Connect the port or set the secret |
| Page loads, says the API is unreachable | The API container is still starting | Wait a minute and reload |
| Payroll panel says not connected | `payroll-api` is not deployed or not connected to the UI | Expected until the stretch goal |
| Error on `max_days_per_request` | The value is outside the allowed range | Read the message; it says the range |

## Part 3: Massdriver Architect (0:33)

Watch only. The presenter describes an unrelated app to Claude Code in one sentence and the `massdriver:architect` plugin produces a bundle that fits this catalog: the dependencies are the same resource types you connected in the lab, and the app's environment variables are already wired from them. Then it is published and dropped onto a canvas like any other bundle.

The plugin is at [massdriver-cloud/claude-plugins](https://github.com/massdriver-cloud/claude-plugins). If you installed Claude Code and the plugin before the session, you can run the same prompt on your own machine afterward and compare. You can also generate your own application on Kafka and MariaDB. Nothing later depends on it.

## Part 4: reading a bundle (0:43)

The presenter reads `bundles/timeoff-api/massdriver.yaml` in this repo top to bottom. Open it yourself. For each block, two questions: what does it do in the code, and what does it do on the canvas.

| Block | On the canvas | In the code |
|-------|---------------|-------------|
| `params` | The form. Presets, ranges, readable error messages, dropdowns fed from a connected resource (`$md.enum`), fields that only appear when another is set. | Variables, generated by `mass bundle build`. This is basic syncing with your IaC tool. |
| `dependencies` | The ports on the left, typed. Required ones block deploy until connected. | Typed variables shaped like the resource type: `var.mariadb.auth.hostname`. |
| `resources` | The ports on the right. These are your outputs. | `massdriver_resource` blocks whose JSON must match the type. |
| `app.envs` | Nothing visible. | Environment variables built with JQ from params and dependencies: `DATABASE_URL` is `.dependencies.mariadb.dsn`. |
| `app.secrets` | The Secrets section of the form. Required ones block deploy. | Injected as environment variables. |
| `ui` | Field order, widgets, help text. | Nothing. |

Two things worth noticing. `$md.sensitive` on a resource type field is why passwords render as `[SENSITIVE]` everywhere. And the word `dependencies` appears twice with two meanings: the top-level block is Massdriver's, the one inside `params` is JSON Schema's and controls conditional fields. Everyone confuses them once.

## Part 5: release channels (1:05)

Every bundle version is semver and immutable once published. An instance pins either an exact version or a channel: `~1` means any 1.x, `~1.1` means any 1.1.x patch, `latest` means the highest stable. When a matching version is published, the instance redeploys on its own.

Everyone upgrades their own bundles.

- Click your `timeoff-api` instance, **Edit Version**, **Release Channels**, pick `~1`, save. Saving deploys.
- Leave your release strategy on **stable**.

You publish 1.0.1. Your instance sees a deploy start on its own, and a small change lands in the page. An instance still pinned to an exact `1.0.0` sees an available upgrade and nothing else.

Then you publish a development release, `1.1.0-dev.<timestamp>`. Nothing moves, because you are on stable. Flip your release strategy to **development** and pick it up; a new request type appears in your form.

The pattern to take home: production on `~1.2` (patches flow, features do not), staging on `~1` with the development strategy, personal sandboxes on `latest` with the development strategy.

## After the session

### Build your own bundle

Every bundle here is a folder with a `massdriver.yaml`, a `src/` directory, an `operator.md` runbook, and an icon. Copy one, change the image and the parameters, and publish it:

```sh
brew install massdriver
export MASSDRIVER_API_KEY=<a key from Settings, Service Accounts, in your org>
export MASSDRIVER_ORGANIZATION_ID=<your org id>
cd bundles/timeoff-api
mass bundle build
mass bundle lint
mass bundle publish --development
```

Your bundle appears in your organization's bundle list and works on the canvas like the others.

Default bundle access in your organization is already set to **All projects**, so anything you publish is usable by every project you make.

### Run the example apps locally

The application code is under `example/apps`, with a `docker-compose.yml` that runs the whole stack on a laptop:

```sh
cd example
docker compose up --build
open http://localhost:8000/local/
```

### What is in this catalog

| Bundle | Connects to | What it does | Image |
|--------|-------------|--------------|-------|
| `mariadb` | landing zone | One database from your landing zone as a `mariadb-authentication` resource | |
| `kafka` | landing zone | Your topics as a `kafka-authentication` resource | |
| `timeoff-api` | `mariadb`, `kafka` | Requests and approvals. Publishes a decision event per approval or denial. Emits its endpoint. | [massdrivercloud/hr-workshop-timeoff-api](https://hub.docker.com/r/massdrivercloud/hr-workshop-timeoff-api) |
| `payroll-api` | `kafka` | Consumes decision events into a PTO ledger. Emits its endpoint. | [massdrivercloud/hr-workshop-payroll-api](https://hub.docker.com/r/massdrivercloud/hr-workshop-payroll-api) |
| `timeoff-ui` | `timeoff-api`, `payroll-api` (optional) | The web page. Reads both API endpoints from its dependencies. Emits the site URL. | [massdrivercloud/hr-workshop-timeoff-ui](https://hub.docker.com/r/massdrivercloud/hr-workshop-timeoff-ui) |

Your applications answer at:

```
http://<your hostname>/              timeoff-ui
http://<your hostname>/api/timeoff/  timeoff-api
http://<your hostname>/api/payroll/  payroll-api
```

### Things to know

- Your database credential reaches your databases and nothing else. Your Kafka credential reaches your topics and nothing else. The servers enforce this.
- Each landing zone has limits on database connections, Kafka throughput, and compute. Hitting one gives your application an error; it does not affect anyone else.
- Every deploy runs a plan and a security scan first. The Deployments tab has the logs.
