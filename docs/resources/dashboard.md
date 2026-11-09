---
page_title: "lucida_dashboard (Resource)"
subcategory: ""
description: |-
  A dashboard with its tiles; tiles name their insight.
---

# lucida_dashboard (Resource)

A dashboard with its tiles; tiles name their insight.

Manages the `dashboards` collection, the same kind `lucida pull|plan|apply` calls `dashboards`.

## Example Usage

```terraform
resource "lucida_dashboard" "example" {
  name = "Growth"
  definition = jsonencode({ description = "Weekly growth", tiles = [{ insight = "Weekly signups", position = 0 }] })
}
```

## Schema

### Required

- `name` (String) Natural key. Changing it replaces the resource.
- `definition` (String) JSON object of the managed fields: `description`, `filters`, `tags`, `pinned`, `tiles`.

### Read-Only

- `id` (String) The Lucida id (the slug for endpoints).

## Drift

On every refresh the fields named in `definition` are compared with Lucida; a change made in the UI shows as a diff. Fields the definition leaves out are never compared.

## Import

Import takes the Lucida id (the slug for endpoints):

```shell
terraform import lucida_dashboard.example <id>
```
