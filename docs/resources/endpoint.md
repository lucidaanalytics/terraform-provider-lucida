---
page_title: "lucida_endpoint (Resource)"
subcategory: ""
description: |-
  A query endpoint, keyed by slug; a change to query or variables creates a new version.
---

# lucida_endpoint (Resource)

A query endpoint, keyed by slug; a change to query or variables creates a new version.

Manages the `endpoints` collection, the same kind `lucida pull|plan|apply` calls `endpoints`.

## Example Usage

```terraform
resource "lucida_endpoint" "example" {
  slug = "top-pages"
  definition = jsonencode({ name = "Top pages", refresh_every_minutes = 60, query = { kind = "trend", series = [{ event = "$pageview" }] } })
}
```

## Schema

### Required

- `slug` (String) Natural key. Changing it replaces the resource.
- `definition` (String) JSON object of the managed fields: `name`, `description`, `refresh_every_minutes`, `query`, `variables`. Changing `slug` replaces the resource.

### Read-Only

- `id` (String) The Lucida id (the slug for endpoints).

## Drift

On every refresh the fields named in `definition` are compared with Lucida; a change made in the UI shows as a diff. Fields the definition leaves out are never compared.

## Import

Import takes the Lucida id (the slug for endpoints):

```shell
terraform import lucida_endpoint.example <id>
```
