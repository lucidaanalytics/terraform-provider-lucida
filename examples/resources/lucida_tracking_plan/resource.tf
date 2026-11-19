resource "lucida_tracking_plan" "example" {
  name = "Core events"
  definition = jsonencode({ mode = "warn", schema = { events = [{ name = "signed_up", properties = [{ name = "plan", required = true }] }] } })
}
