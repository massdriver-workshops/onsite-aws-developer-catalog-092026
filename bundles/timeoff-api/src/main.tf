locals {
  params = {
    image                = var.image
    replicas             = var.replicas
    resources            = var.resources
    path                 = var.path
    events_topic         = var.events_topic
    max_days_per_request = var.max_days_per_request
    approval             = var.approval
  }
  dependencies = {
    landing_zone = var.landing_zone
    mariadb      = var.mariadb
    kafka        = var.kafka
  }
  # Written next to massdriver.yaml by the provisioner from the instance's secrets.
  secrets = try(jsondecode(file("${path.root}/../secrets.json")), {})

  app_block = try(yamldecode(file("${path.root}/../massdriver.yaml")).app, { envs = {}, secrets = {} })

  # Both keys point at the same object so the JQ in massdriver.yaml can use either name.
  jq_inputs = jsonencode({
    params       = local.params
    dependencies = local.dependencies
    connections  = local.dependencies
    secrets      = local.secrets
  })

  envs = { for name, _ in local.app_block.envs : name => tostring(jsondecode(data.jq_query.envs[name].result)) }

  ingress = var.landing_zone.kubernetes.ingress
  url     = "${local.ingress.scheme}://${local.ingress.hostname}${var.path}/"
}

data "jq_query" "envs" {
  for_each = local.app_block.envs
  data     = local.jq_inputs
  query    = each.value
}

resource "helm_release" "app" {
  name             = var.md_metadata.name_prefix
  namespace        = var.landing_zone.kubernetes.namespace
  chart            = "${path.module}/chart"
  create_namespace = false
  wait             = true
  timeout          = 300

  values = [yamlencode({
    name     = var.md_metadata.name_prefix
    labels   = var.md_metadata.default_tags
    image    = "${var.image.repository}:${var.image.tag}"
    replicas = var.replicas
    port     = 8080
    resources = {
      cpu      = var.resources.cpu
      memoryMb = var.resources.memory_mb
    }
    env = local.envs
    ingress = {
      className = local.ingress.class_name
      host      = local.ingress.hostname
      path      = var.path
    }
  })]

  dynamic "set_sensitive" {
    for_each = local.secrets
    content {
      name  = "secrets.${set_sensitive.key}"
      value = set_sensitive.value
    }
  }
}

output "url" {
  description = "Where the API answers. GET <url>info returns version and policy."
  value       = local.url
}
