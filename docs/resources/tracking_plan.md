---
page_title: "lucida_tracking_plan (Resource)"
subcategory: ""
description: |-
  A tracking plan (event schema and enforcement mode `off|warn|enforce`), keyed by name.
---

# lucida_tracking_plan (Resource)

A tracking plan (event schema and enforcement mode `off|warn|enforce`), keyed by name.

Manages the `tracking-plans` collection, the same kind `lucida pull|plan|apply` calls `tracking-plans`.

## Example Usage

```terraform
resource "lucida_tracking_plan" "example" {
  name = "Core events"
  definition = jsonencode({ mode = "warn", schema = { events = [{ name = "signed_up", properties = [{ name = "plan", required = true }] }] } })
}
```

## Schema

### Required

- `name` (String) Natural key. Changing it replaces the resource.
- `definition` (String) JSON object of the managed fields: `mode`, `schema`.

### Read-Only

- `id` (String) The Lucida id (the slug for endpoints).

## Drift

On every refresh the fields named in `definition` are compared with Lucida; a change made in the UI shows as a diff. Fields the definition leaves out are never compared.

## Import

Import takes the Lucida id (the slug for endpoints):

```shell
terraform import lucida_tracking_plan.example <id>
```
