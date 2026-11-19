---
page_title: "lucida_sdk_settings (Resource)"
subcategory: ""
description: |-
  The `sdk` settings section (a singleton: set only the keys you manage; destroy only removes it from state).
---

# lucida_sdk_settings (Resource)

The `sdk` settings section (a singleton: set only the keys you manage; destroy only removes it from state).

Manages the `settings` collection, the same kind `lucida pull|plan|apply` calls `settings`.

## Example Usage

```terraform
resource "lucida_sdk_settings" "example" {
  definition = jsonencode({ consent_mode = "opt_in", respect_gpc = true })
}
```

## Schema

### Required

- `definition` (String) JSON object of the managed fields: any key of the object (the API's own names).

### Read-Only

- `id` (String) The Lucida id (the slug for endpoints).
- `section` (String) Fixed key of this singleton: `sdk`.

## Drift

On every refresh the fields named in `definition` are compared with Lucida; a change made in the UI shows as a diff. Fields the definition leaves out are never compared.

## Import

Import takes the Lucida id (the slug for endpoints):

```shell
terraform import lucida_sdk_settings.example <id>
```
