resource "lucida_dashboard" "example" {
  name = "Growth"
  definition = jsonencode({ description = "Weekly growth", tiles = [{ insight = "Weekly signups", position = 0 }] })
}
