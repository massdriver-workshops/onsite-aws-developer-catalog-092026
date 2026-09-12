# The page's address, so it is one click away on the canvas.
resource "massdriver_resource" "site" {
  field = "site"
  name  = "Time Off at ${local.ingress.hostname}"
  resource = jsonencode({
    url      = local.url
    scheme   = local.ingress.scheme
    hostname = local.ingress.hostname
    path     = ""
    version  = var.image.tag
  })
}
