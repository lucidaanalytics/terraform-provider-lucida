resource "lucida_experiment" "example" {
  name = "checkout-button-colour"
  definition = jsonencode({ hypothesis = "A green button lifts checkout starts", kind = "feature", variants = [{ key = "control" }, { key = "green" }], traffic_pct = 50 })
}
