---
page_title: "lucida_metric (Resource)"
subcategory: ""
description: |-
  A metric definition, keyed by name.
---

# lucida_metric (Resource)

A metric definition, keyed by name.

Manages the `metrics` collection, the same kind `lucida pull|plan|apply` calls `metrics`.

## Example Usage

```terraform
resource "lucida_metric" "example" {
  name = "checkout_conversion"
  definition = jsonencode({ display_name = "Checkout conversion", unit = "ratio", kind = "funnel", definition = { steps = ["cart_viewed", "order_completed"] } })
}
```

## Schema

### Required

- `name` (String) Natural key. Changing it replaces the resource.
- `definition` (String) JSON object of the managed fields: `display_name`, `description`, `unit`, `kind`, `definition`, `tags`.

### Read-Only

- `id` (String) The Lucida id (the slug for endpoints).

## Drift

On every refresh the fields named in `definition` are compared with Lucida; a change made in the UI shows as a diff. Fields the definition leaves out are never compared.

## Import

Import takes the Lucida id (the slug for endpoints):

```shell
terraform import lucida_metric.example <id>
```
