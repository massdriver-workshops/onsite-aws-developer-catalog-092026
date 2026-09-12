locals {
  params = {
    image            = var.image
    replicas         = var.replicas
    resources        = var.resources
    timeoff_api_path = var.timeoff_api_path
    payroll_api_path = var.payroll_api_path
  }
  dependencies = {
    landing_zone = var.landing_zone
  }

  app_block = try(yamldecode(file("${path.root}/../massdriver.yaml")).app, { envs = {} })

  # Both keys point at the same object so the JQ in massdriver.yaml can use either name.
  jq_inputs = jsonencode({
    params       = local.params
    dependencies = local.dependencies
    connections  = local.dependencies
    secrets      = {}
  })

  envs = { for name, _ in local.app_block.envs : name => tostring(jsondecode(data.jq_query.envs[name].result)) }

  ingress = var.landing_zone.kubernetes.ingress
  url     = "${local.ingress.scheme}://${local.ingress.hostname}/"
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
      path      = "/"
    }
    # nginx renders its config and caches under these at start; the root stays read-only.
    writableDirs = ["/tmp", "/etc/nginx/conf.d"]
  })]
}

output "url" {
  description = "Open this in a browser."
  value       = local.url
}
