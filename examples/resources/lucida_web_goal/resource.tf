resource "lucida_web_goal" "example" {
  name = "Signed up"
  definition = jsonencode({ kind = "event", target = "signed_up", value = 10 })
}
