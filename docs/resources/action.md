---
page_title: "lucida_action (Resource)"
subcategory: ""
description: |-
  A named group of event steps (an action), keyed by name.
---

# lucida_action (Resource)

A named group of event steps (an action), keyed by name.

Manages the `actions` collection, the same kind `lucida pull|plan|apply` calls `actions`.

## Example Usage

```terraform
resource "lucida_action" "example" {
  name = "signed_up"
  definition = jsonencode({ description = "A visitor created an account", steps = [{ event = "signed_up" }] })
}
```

## Schema

### Required

- `name` (String) Natural key. Changing it replaces the resource.
- `definition` (String) JSON object of the managed fields: `description`, `steps`.

### Read-Only

- `id` (String) The Lucida id (the slug for endpoints).

## Drift

On every refresh the fields named in `definition` are compared with Lucida; a change made in the UI shows as a diff. Fields the definition leaves out are never compared.

## Import

Import takes the Lucida id (the slug for endpoints):

```shell
terraform import lucida_action.example <id>
```
