resource "lucida_endpoint" "example" {
  slug = "top-pages"
  definition = jsonencode({ name = "Top pages", refresh_every_minutes = 60, query = { kind = "trend", series = [{ event = "$pageview" }] } })
}
