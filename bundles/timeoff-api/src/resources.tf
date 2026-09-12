resource "massdriver_resource" "api" {
  field = "api"
  name  = "timeoff-api at ${local.ingress.hostname}${var.path}"
  resource = jsonencode({
    url      = local.url
    scheme   = local.ingress.scheme
    hostname = local.ingress.hostname
    path     = var.path
    version  = var.image.tag
  })
}
