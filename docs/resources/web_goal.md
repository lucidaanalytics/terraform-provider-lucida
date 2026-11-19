---
page_title: "lucida_web_goal (Resource)"
subcategory: ""
description: |-
  A web goal, keyed by name. The API cannot edit a goal: changing anything but the name replaces it.
---

# lucida_web_goal (Resource)

A web goal, keyed by name. The API cannot edit a goal: changing anything but the name replaces it.

Manages the `web-goals` collection, the same kind `lucida pull|plan|apply` calls `web-goals`.

## Example Usage

```terraform
resource "lucida_web_goal" "example" {
  name = "Signed up"
  definition = jsonencode({ kind = "event", target = "signed_up", value = 10 })
}
```

## Schema

### Required

- `name` (String) Natural key. Changing it replaces the resource.
- `definition` (String) JSON object of the managed fields: `kind`, `target`, `value`, `value_property`, `timestamp_field`, `distinct_id_field`. Changing `kind`, `target`, `value`, `value_property`, `timestamp_field`, `distinct_id_field` replaces the resource.

### Read-Only

- `id` (String) The Lucida id (the slug for endpoints).

## Drift

On every refresh the fields named in `definition` are compared with Lucida; a change made in the UI shows as a diff. Fields the definition leaves out are never compared.

## Import

Import takes the Lucida id (the slug for endpoints):

```shell
terraform import lucida_web_goal.example <id>
```
