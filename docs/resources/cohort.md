---
page_title: "lucida_cohort (Resource)"
subcategory: ""
description: |-
  A cohort, keyed by name. `kind` is fixed at create.
---

# lucida_cohort (Resource)

A cohort, keyed by name. `kind` is fixed at create.

Manages the `cohorts` collection, the same kind `lucida pull|plan|apply` calls `cohorts`.

## Example Usage

```terraform
resource "lucida_cohort" "example" {
  name = "power_users"
  definition = jsonencode({ kind = "dynamic", filters = { match = "all", conditions = [{ event = "report_viewed", op = "gte", count = 5, days = 30 }] } })
}
```

## Schema

### Required

- `name` (String) Natural key. Changing it replaces the resource.
- `definition` (String) JSON object of the managed fields: `description`, `kind`, `filters`. Changing `kind` replaces the resource.

### Read-Only

- `id` (String) The Lucida id (the slug for endpoints).

## Drift

On every refresh the fields named in `definition` are compared with Lucida; a change made in the UI shows as a diff. Fields the definition leaves out are never compared.

## Import

Import takes the Lucida id (the slug for endpoints):

```shell
terraform import lucida_cohort.example <id>
```
