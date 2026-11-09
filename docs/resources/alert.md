---
page_title: "lucida_alert (Resource)"
subcategory: ""
description: |-
  An insight alert, keyed by name; `insight` names the insight it watches.
---

# lucida_alert (Resource)

An insight alert, keyed by name; `insight` names the insight it watches.

Manages the `alerts` collection, the same kind `lucida pull|plan|apply` calls `alerts`.

## Example Usage

```terraform
resource "lucida_alert" "example" {
  name = "Signups drop"
  definition = jsonencode({ insight = "Weekly signups", condition = { kind = "threshold", op = "lt", value = 100 }, check_interval = "1h", enabled = true })
}
```

## Schema

### Required

- `name` (String) Natural key. Changing it replaces the resource.
- `definition` (String) JSON object of the managed fields: `insight`, `condition`, `check_interval`, `channels`, `enabled`. Changing `insight` replaces the resource.

### Read-Only

- `id` (String) The Lucida id (the slug for endpoints).

## Drift

On every refresh the fields named in `definition` are compared with Lucida; a change made in the UI shows as a diff. Fields the definition leaves out are never compared.

## Import

Import takes the Lucida id (the slug for endpoints):

```shell
terraform import lucida_alert.example <id>
```
