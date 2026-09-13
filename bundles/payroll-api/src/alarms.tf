# Registers the alarm with the instance so it shows on the Alarms tab and in the
# dependency graph. State changes arrive from a monitor posting to the instance's
# alarm webhook; this bundle deploys no monitor of its own.
resource "massdriver_instance_alarm" "replicas_unavailable" {
  display_name        = "Replicas unavailable"
  cloud_resource_id   = "kubernetes:${var.landing_zone.kubernetes.namespace}/${var.md_metadata.name_prefix}:replicas-unavailable"
  comparison_operator = "GREATER_THAN"
  threshold           = 0
  period              = 300

  metric {
    name      = "kube_deployment_status_replicas_unavailable"
    namespace = "kubernetes"
    statistic = "Maximum"
    dimensions = {
      namespace  = var.landing_zone.kubernetes.namespace
      deployment = var.md_metadata.name_prefix
    }
  }
}
