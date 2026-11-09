resource "lucida_destination" "example" {
  name = "slack-alerts"
  definition = jsonencode({ kind = "slack", config = { channel = "#growth" } })
  credentials = jsonencode({ token = var.lucida_token }) # sensitive; sent on create only
}
