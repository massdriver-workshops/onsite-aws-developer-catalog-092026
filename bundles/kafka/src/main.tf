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
  kafka = var.landing_zone.kafka
}

resource "massdriver_resource" "kafka" {
  field = "kafka"
  name  = "Kafka ${local.kafka.topic_prefix}*"
  resource = jsonencode({
    bootstrap_brokers = local.kafka.bootstrap_brokers
    auth = {
      mechanism = local.kafka.sasl.mechanism
      username  = local.kafka.sasl.username
      password  = local.kafka.sasl.password
      tls       = local.kafka.tls.enabled
    }
    topics                = local.kafka.topics
    consumer_group_prefix = local.kafka.consumer_group_prefix
  })
}
