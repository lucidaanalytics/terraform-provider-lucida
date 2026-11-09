resource "lucida_action" "example" {
  name = "signed_up"
  definition = jsonencode({ description = "A visitor created an account", steps = [{ event = "signed_up" }] })
}
