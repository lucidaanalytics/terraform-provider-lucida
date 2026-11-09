---
page_title: "lucida_destination (Resource)"
subcategory: ""
description: |-
  A destination connector (the same API object as `lucida_connector`; pick a destination `kind`).
---

# lucida_destination (Resource)

A destination connector (the same API object as `lucida_connector`; pick a destination `kind`).

Manages the `connectors` collection, the same kind `lucida pull|plan|apply` calls `connectors`.

## Example Usage

```terraform
resource "lucida_destination" "example" {
  name = "slack-alerts"
  definition = jsonencode({ kind = "slack", config = { channel = "#growth" } })
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
terraform import lucida_destination.example <id>
```
