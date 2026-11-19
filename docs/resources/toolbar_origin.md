---
page_title: "lucida_toolbar_origin (Resource)"
subcategory: ""
description: |-
  The origins the toolbar may run on (a singleton: create adopts the existing list, destroy only removes it from state).
---

# lucida_toolbar_origin (Resource)

The origins the toolbar may run on (a singleton: create adopts the existing list, destroy only removes it from state).

Manages the `toolbar-origins` collection, the same kind `lucida pull|plan|apply` calls `toolbar-origins`.

## Example Usage

```terraform
resource "lucida_toolbar_origin" "example" {
  definition = jsonencode({ origins = ["https://app.example.com"] })
}
```

## Schema

### Required

- `definition` (String) JSON object of the managed fields: `origins`.

### Read-Only

- `id` (String) The Lucida id (the slug for endpoints).
- `name` (String) Fixed key of this singleton: `origins`.

## Drift

On every refresh the fields named in `definition` are compared with Lucida; a change made in the UI shows as a diff. Fields the definition leaves out are never compared.

## Import

Import takes the Lucida id (the slug for endpoints):

```shell
terraform import lucida_toolbar_origin.example <id>
```
