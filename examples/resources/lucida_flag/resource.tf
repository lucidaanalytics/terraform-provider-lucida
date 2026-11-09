resource "lucida_flag" "example" {
  key = "new-checkout"
  definition = jsonencode({ name = "New checkout", enabled = true, filters = { rollout = 25 } })
}
