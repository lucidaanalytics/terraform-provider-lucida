resource "lucida_ingest_settings" "example" {
  definition = jsonencode({ capture_allowed_origins = ["https://app.example.com"], server_event_consent = "drop_denied" })
}
