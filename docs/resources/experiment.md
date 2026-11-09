---
page_title: "lucida_experiment (Resource)"
subcategory: ""
description: |-
  An experiment, keyed by name. Destroying archives it (the API has no delete).
---

# lucida_experiment (Resource)

An experiment, keyed by name. Destroying archives it (the API has no delete).

Manages the `experiments` collection, the same kind `lucida pull|plan|apply` calls `experiments`.

## Example Usage

```terraform
resource "lucida_experiment" "example" {
  name = "checkout-button-colour"
  definition = jsonencode({ hypothesis = "A green button lifts checkout starts", kind = "feature", variants = [{ key = "control" }, { key = "green" }], traffic_pct = 50 })
}
```

## Schema

### Required

- `name` (String) Natural key. Changing it replaces the resource.
- `definition` (String) JSON object of the managed fields: `hypothesis`, `kind`, `variants`, `audience`, `traffic_pct`, `schedule`, `exposure`, `metrics`, `stats_config`, `requires_approval`. Changing `kind` replaces the resource.

### Read-Only

- `id` (String) The Lucida id (the slug for endpoints).

## Drift

On every refresh the fields named in `definition` are compared with Lucida; a change made in the UI shows as a diff. Fields the definition leaves out are never compared.

## Import

Import takes the Lucida id (the slug for endpoints):

```shell
terraform import lucida_experiment.example <id>
```
