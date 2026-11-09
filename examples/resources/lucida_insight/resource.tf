resource "lucida_insight" "example" {
  name = "Weekly signups"
  definition = jsonencode({ description = "Signups per week", query = { kind = "trend", series = [{ event = "signed_up" }], interval = "week" } })
}
