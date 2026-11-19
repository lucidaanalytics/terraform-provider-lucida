resource "lucida_sdk_settings" "example" {
  definition = jsonencode({ consent_mode = "opt_in", respect_gpc = true })
}
