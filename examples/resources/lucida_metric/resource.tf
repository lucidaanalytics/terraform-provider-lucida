resource "lucida_metric" "example" {
  name = "checkout_conversion"
  definition = jsonencode({ display_name = "Checkout conversion", unit = "ratio", kind = "funnel", definition = { steps = ["cart_viewed", "order_completed"] } })
}
