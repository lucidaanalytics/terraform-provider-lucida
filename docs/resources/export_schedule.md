---
page_title: "lucida_export_schedule (Resource)"
subcategory: ""
description: |-
  A batch export schedule. Where and what it exports is fixed at create; `credentials` are sent on create only.
---

# lucida_export_schedule (Resource)

A batch export schedule. Where and what it exports is fixed at create; `credentials` are sent on create only.

Manages the `export-schedules` collection, the same kind `lucida pull|plan|apply` calls `export-schedules`.

## Example Usage

```terraform
resource "lucida_export_schedule" "example" {
  name = "daily-events"
  definition = jsonencode({ stream = "events", format = "parquet", destination = "s3", interval_secs = 86400 })
  credentials = jsonencode({ token = var.lucida_token }) # sensitive; sent on create only
}
```

## Schema

### Required

- `name` (String) Natural key. Changing it replaces the resource.
- `definition` (String) JSON object of the managed fields: `stream`, `format`, `destination`, `warehouse`, `interval_secs`, `chunk_secs`, `options`, `enabled`. Changing `stream`, `format`, `destination`, `warehouse`, `chunk_secs`, `options` replaces the resource.

### Optional

- `credentials` (String, Sensitive) JSON object, sent on create only and never read back. Rotate it with `terraform apply -replace`.

### Read-Only

- `id` (String) The Lucida id (the slug for endpoints).

## Drift

On every refresh the fields named in `definition` are compared with Lucida; a change made in the UI shows as a diff. Fields the definition leaves out are never compared.

## Import

Import takes the Lucida id (the slug for endpoints):

```shell
terraform import lucida_export_schedule.example <id>
```
