resource "lucida_traffic_exclusion" "example" {
  note = "Office network"
  definition = jsonencode({ kind = "ip", value = "203.0.113.0/24", enabled = true })
}
