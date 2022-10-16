###############################################################################
# Variables
###############################################################################

variable "google_api_key" {
  default     = ""
  description = "API key for connecting to the Google Geo API"
  type        = string
}

variable "kubernetes_version" {
  default     = "1.23.12"
  description = "Version of Kubernetes to use in the AKS cluster"
  type        = string
}

variable "subscription_id" {
  description = "ID of Azure subscription into which to deploy the AKS cluster"
  type        = string
}

variable "tenant_id" {
  description = "Azure tenant ID"
  type        = string
}


###############################################################################
# Providers
###############################################################################

provider "azurerm" {
  subscription_id         = var.subscription_id
  tenant_id               = var.tenant_id
  environment             = "public"
  features {}
}

provider "kubernetes" {
  host                   = azurerm_kubernetes_cluster.kubernetes_cluster.kube_config.0.host
  client_certificate     = base64decode(azurerm_kubernetes_cluster.kubernetes_cluster.kube_config.0.client_certificate)
  client_key             = base64decode(azurerm_kubernetes_cluster.kubernetes_cluster.kube_config.0.client_key)
  cluster_ca_certificate = base64decode(azurerm_kubernetes_cluster.kubernetes_cluster.kube_config.0.cluster_ca_certificate)
}


###############################################################################
# Resources
###############################################################################

resource "azurerm_resource_group" "drinksfinder_resource_group" {
  location = "UK South"
  name     = "drinksfinder-rg"
  tags     = {
               Project = "DrinksFinder"
               Owner   = "matthew.geldert@ivanti.com"
               Team    = "found"
             }
}

resource "azurerm_kubernetes_cluster" "kubernetes_cluster" {
  dns_prefix          = "drinksfinder"
  kubernetes_version  = var.kubernetes_version
  location            = "UK South"
  name                = "drinksfinder"
  resource_group_name = azurerm_resource_group.drinksfinder_resource_group.name
  sku_tier            = "Free"
  tags                = {
                          Project = "DrinksFinder"
                          Owner   = "matthew.geldert@ivanti.com"
                          Team    = "found"
                        }

  network_profile {
     network_plugin = "kubenet"
     network_policy = "calico"
     load_balancer_sku = "standard"
  }

  default_node_pool {
    name                = "default"
    type                = "VirtualMachineScaleSets"
    node_count          = 2
    vm_size             = "Standard_B2s"
    os_disk_size_gb     = 30
  }

  identity {
    type = "SystemAssigned"
  }
}

resource "kubernetes_namespace" "drinksfinder" {
  metadata {
    name = "drinksfinder"
  }

  depends_on = [ azurerm_kubernetes_cluster.kubernetes_cluster ]
}

resource "kubernetes_secret" "google_api_key" {
  metadata {
    name = "drinksfinder-google-api-key"
    namespace = "drinksfinder"
  }

  data = {
    google_api_key = var.google_api_key
  }
}
