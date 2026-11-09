resource "lucida_connector" "example" {
  name = "warehouse-sync"
  definition = jsonencode({ kind = "postgres", config = { host = "db.internal", database = "app" }, sync_interval_secs = 900 })
  credentials = jsonencode({ token = var.lucida_token }) # sensitive; sent on create only
}
