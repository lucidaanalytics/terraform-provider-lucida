terraform {
  required_providers {
    lucida = {
      source = "lucidaanalytics/lucida"
    }
  }
}

provider "lucida" {
  # host       = "https://lucidace.com" # or $LUCIDA_HOST
  # api_key    = var.lucida_api_key     # or $LUCIDA_API_KEY (sensitive)
  project_id = var.lucida_project_id    # or $LUCIDA_PROJECT_ID
}
