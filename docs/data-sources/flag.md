---
page_title: "lucida_flag (Data Source)"
subcategory: ""
description: |-
  Looks up a feature flag by key.
---

# lucida_flag (Data Source)

Looks up a feature flag by key, for wiring ids into other resources.

## Example Usage

```terraform
data "lucida_flag" "checkout" {
  key = "new-checkout"
}
```

## Schema

### Required

- `key` (String)

### Read-Only

- `id` (String)
- `name` (String)
- `enabled` (Boolean)
- `definition` (String) JSON of the flag's managed fields, as the resource would hold it.
