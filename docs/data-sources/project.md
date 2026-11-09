---
page_title: "lucida_project (Data Source)"
subcategory: ""
description: |-
  Looks up a project.
---

# lucida_project (Data Source)

Looks up a project; defaults to the provider's project.

## Example Usage

```terraform
data "lucida_project" "this" {}
```

## Schema

### Optional

- `id` (String) Project id (default: the provider's `project_id`).

### Read-Only

- `name` (String)
