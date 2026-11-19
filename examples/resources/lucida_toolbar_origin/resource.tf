resource "lucida_toolbar_origin" "example" {
  definition = jsonencode({ origins = ["https://app.example.com"] })
}
