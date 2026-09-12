terraform {
  required_version = ">= 1.6"
  required_providers {
    massdriver = {
      source  = "massdriver-cloud/massdriver"
      version = "~> 2.0"
    }
  }
}

locals {
  db  = var.landing_zone.mariadb
  dsn = "mysql://${urlencode(local.db.username)}:${urlencode(local.db.password)}@${local.db.hostname}:${local.db.port}/${var.database}?tls=true"
}

resource "massdriver_resource" "mariadb" {
  field = "mariadb"
  name  = "MariaDB ${var.database}"
  resource = jsonencode({
    auth = {
      hostname = local.db.hostname
      port     = local.db.port
      database = var.database
      username = local.db.username
      password = local.db.password
    }
    dsn = local.dsn
    tls = {
      required  = local.db.tls.required
      ca_bundle = local.db.tls.ca_bundle
    }
  })
}
