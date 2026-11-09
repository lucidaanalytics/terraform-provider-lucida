resource "lucida_alert" "example" {
  name = "Signups drop"
  definition = jsonencode({ insight = "Weekly signups", condition = { kind = "threshold", op = "lt", value = 100 }, check_interval = "1h", enabled = true })
}
