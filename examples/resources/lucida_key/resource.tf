resource "lucida_key" "example" {
  name = "ci-read"
  definition = jsonencode({ kind = "project", scopes = ["read"] })
}
