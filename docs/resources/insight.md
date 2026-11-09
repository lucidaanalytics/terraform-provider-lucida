---
page_title: "lucida_insight (Resource)"
subcategory: ""
description: |-
  A saved insight, keyed by name.
---

# lucida_insight (Resource)

A saved insight, keyed by name.

Manages the `insights` collection, the same kind `lucida pull|plan|apply` calls `insights`.

## Example Usage

```terraform
resource "lucida_insight" "example" {
  name = "Weekly signups"
  definition = jsonencode({ description = "Signups per week", query = { kind = "trend", series = [{ event = "signed_up" }], interval = "week" } })
}
```

## Schema

### Required

- `name` (String) Natural key. Changing it replaces the resource.
- `definition` (String) JSON object of the managed fields: `description`, `query`, `settings`, `tags`, `folder`.

### Read-Only

- `id` (String) The Lucida id (the slug for endpoints).

## Drift

On every refresh the fields named in `definition` are compared with Lucida; a change made in the UI shows as a diff. Fields the definition leaves out are never compared.

## Import

Import takes the Lucida id (the slug for endpoints):

```shell
terraform import lucida_insight.example <id>
```
