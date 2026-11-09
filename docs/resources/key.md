---
page_title: "lucida_key (Resource)"
subcategory: ""
description: |-
  A project API key. The secret is returned once as the sensitive `secret` attribute; destroying revokes the key.
---

# lucida_key (Resource)

A project API key. The secret is returned once as the sensitive `secret` attribute; destroying revokes the key.

Manages the `keys` collection, the same kind `lucida pull|plan|apply` calls `keys`.

## Example Usage

```terraform
resource "lucida_key" "example" {
  name = "ci-read"
  definition = jsonencode({ kind = "project", scopes = ["read"] })
}
```

## Schema

### Required

- `name` (String) Natural key. Changing it replaces the resource.
- `definition` (String) JSON object of the managed fields: `kind`, `scopes`, `revoked`. Created once: any change replaces the resource.

### Optional


### Read-Only

- `id` (String) The Lucida id (the slug for endpoints).
- `secret` (String, Sensitive) The key secret, returned once at create. It is stored in Terraform state: protect the state.

## Drift

On every refresh the fields named in `definition` are compared with Lucida; a change made in the UI shows as a diff. Fields the definition leaves out are never compared.

## Import

Import takes the Lucida id (the slug for endpoints):

```shell
terraform import lucida_key.example <id>
```
