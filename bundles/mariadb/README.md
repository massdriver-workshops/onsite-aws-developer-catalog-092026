# mariadb

Turns one of the databases in your landing zone into a resource your application can depend on.

Add it to the canvas, pick a database, deploy. It emits a `mariadb-authentication` resource: host, port, database, username, password, a ready-made connection URL, and the CA bundle your driver needs to verify the server. Connect that to any application bundle with a MariaDB port.

It creates nothing. The database and the credential already exist; this bundle shapes them for an application.
