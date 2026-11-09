resource "lucida_cohort" "example" {
  name = "power_users"
  definition = jsonencode({ kind = "dynamic", filters = { match = "all", conditions = [{ event = "report_viewed", op = "gte", count = 5, days = 30 }] } })
}
