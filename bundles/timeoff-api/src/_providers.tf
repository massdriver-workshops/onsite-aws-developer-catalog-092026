terraform {
  required_version = ">= 1.6"
  required_providers {
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.17"
    }
    jq = {
      source  = "massdriver-cloud/jq"
      version = "~> 0.2"
    }
    massdriver = {
      source  = "massdriver-cloud/massdriver"
      version = "~> 2.0"
    }
  }
}

# The landing zone carries a credential that can act only inside its own
# namespace, which is all this bundle needs.
provider "helm" {
  kubernetes {
    host                   = var.landing_zone.kubernetes.cluster.server
    cluster_ca_certificate = base64decode(var.landing_zone.kubernetes.cluster.certificate_authority_data)
    token                  = var.landing_zone.kubernetes.cluster.token
  }
}
