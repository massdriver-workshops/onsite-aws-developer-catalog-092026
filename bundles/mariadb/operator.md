---
templating: mustache
---

# mariadb

{{#resources.mariadb}}
Database `{{resources.mariadb.auth.database}}` on `{{resources.mariadb.auth.hostname}}:{{resources.mariadb.auth.port}}`. TLS required: {{resources.mariadb.tls.required}}.

## Connect from a laptop

Save the `tls.ca_bundle` field of the resource to `ca.pem`, then:

```sh
mariadb --ssl --ssl-verify-server-cert --ssl-ca ca.pem \
  -h {{resources.mariadb.auth.hostname}} -P {{resources.mariadb.auth.port}} \
  -u {{resources.mariadb.auth.username}} -p {{resources.mariadb.auth.database}}
```

The password is on the resource, masked in the UI, readable by `mass resource get`.
{{/resources.mariadb}}

## Application cannot connect

Almost always TLS. The server refuses plain connections, and the error looks like a bad password. Confirm the client uses the connection URL as given and trusts the CA bundle.

## Access denied on another database

Expected. This credential reaches the databases in your landing zone and nothing else. Add another `mariadb` instance for another database.
