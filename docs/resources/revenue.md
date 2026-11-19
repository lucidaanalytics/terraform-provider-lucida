---
page_title: "lucida_revenue (Resource)"
subcategory: ""
description: |-
  The project's revenue mapping (a singleton: create adopts the saved config, destroy only removes it from state).
---

# lucida_revenue (Resource)

The project's revenue mapping (a singleton: create adopts the saved config, destroy only removes it from state).

Manages the `revenue` collection, the same kind `lucida pull|plan|apply` calls `revenue`.

## Example Usage

```terraform
resource "lucida_revenue" "example" {
  definition = jsonencode({ base_currency = "EUR", minor_units = true, purchase_event = "order_completed" })
}
```

## Schema

### Required

- `definition` (String) JSON object of the managed fields: any key of the object (the API's own names).

### Read-Only

- `id` (String) The Lucida id (the slug for endpoints).
- `name` (String) Fixed key of this singleton: `revenue`.

## Drift

On every refresh the fields named in `definition` are compared with Lucida; a change made in the UI shows as a diff. Fields the definition leaves out are never compared.

## Import

Import takes the Lucida id (the slug for endpoints):

```shell
terraform import lucida_revenue.example <id>
```
