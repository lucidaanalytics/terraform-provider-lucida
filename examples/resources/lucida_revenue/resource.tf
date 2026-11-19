resource "lucida_revenue" "example" {
  definition = jsonencode({ base_currency = "EUR", minor_units = true, purchase_event = "order_completed" })
}
