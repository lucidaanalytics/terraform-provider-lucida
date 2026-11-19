---
page_title: "lucida_ingest_settings (Resource)"
subcategory: ""
description: |-
  The `ingest` settings section (a singleton: set only the keys you manage; destroy only removes it from state).
---

# lucida_ingest_settings (Resource)

The `ingest` settings section (a singleton: set only the keys you manage; destroy only removes it from state).

Manages the `settings` collection, the same kind `lucida pull|plan|apply` calls `settings`.

## Example Usage

```terraform
resource "lucida_ingest_settings" "example" {
  definition = jsonencode({ capture_allowed_origins = ["https://app.example.com"], server_event_consent = "drop_denied" })
}
```

## Schema

### Required

- `definition` (String) JSON object of the managed fields: any key of the object (the API's own names).

### Read-Only

- `id` (String) The Lucida id (the slug for endpoints).
- `section` (String) Fixed key of this singleton: `ingest`.

## Drift

On every refresh the fields named in `definition` are compared with Lucida; a change made in the UI shows as a diff. Fields the definition leaves out are never compared.

## Import

Import takes the Lucida id (the slug for endpoints):

```shell
terraform import lucida_ingest_settings.example <id>
```
