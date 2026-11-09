resource "lucida_export_schedule" "example" {
  name = "daily-events"
  definition = jsonencode({ stream = "events", format = "parquet", destination = "s3", interval_secs = 86400 })
  credentials = jsonencode({ token = var.lucida_token }) # sensitive; sent on create only
}
