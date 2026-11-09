---
page_title: "lucida_connector (Resource)"
subcategory: ""
description: |-
  A connector instance (source or destination), keyed by name. `credentials` are sent on create only.
---

# lucida_connector (Resource)

A connector instance (source or destination), keyed by name. `credentials` are sent on create only.

Manages the `connectors` collection, the same kind `lucida pull|plan|apply` calls `connectors`.

## Example Usage

```terraform
resource "lucida_connector" "example" {
  name = "warehouse-sync"
  definition = jsonencode({ kind = "postgres", config = { host = "db.internal", database = "app" }, sync_interval_secs = 900 })
  credentials = jsonencode({ token = var.lucida_token }) # sensitive; sent on create only
}
```

## Schema

### Required

- `name` (String) Natural key. Changing it replaces the resource.
- `definition` (String) JSON object of the managed fields: `kind`, `config`, `sync_interval_secs`. Changing `kind` replaces the resource.

### Optional

- `credentials` (String, Sensitive) JSON object, sent on create only and never read back. Rotate it with `terraform apply -replace`.

### Read-Only

- `id` (String) The Lucida id (the slug for endpoints).

## Drift

On every refresh the fields named in `definition` are compared with Lucida; a change made in the UI shows as a diff. Fields the definition leaves out are never compared.

## Import

Import takes the Lucida id (the slug for endpoints):

```shell
terraform import lucida_connector.example <id>
```
