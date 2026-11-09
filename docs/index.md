---
page_title: "Lucida Provider"
subcategory: ""
description: |-
  Manage Lucida project definitions as code.
---

# Lucida Provider

Manages a Lucida project's definitions. The resources are generated from the same model as the `lucida` CLI (`lucida pull`, `plan`, `apply`), so Terraform and the CLI see identical kinds and fields.

## Example Usage

```terraform
terraform {
  required_providers {
    lucida = {
      source = "lucidaanalytics/lucida"
    }
  }
}

provider "lucida" {
  # host       = "https://lucidace.com" # or $LUCIDA_HOST
  # api_key    = var.lucida_api_key     # or $LUCIDA_API_KEY (sensitive)
  project_id = var.lucida_project_id    # or $LUCIDA_PROJECT_ID
}
```

## Schema

### Optional

- `host` (String) Lucida API origin. Defaults to `$LUCIDA_HOST` or `https://lucidace.com`.
- `api_key` (String, Sensitive) API key. Defaults to `$LUCIDA_API_KEY`. Use a key scoped to what you manage.
- `project_id` (String) Project every resource lives in. Defaults to `$LUCIDA_PROJECT_ID`.

## Resources

| Resource | Manages |
| --- | --- |
| `lucida_action` | A named group of event steps (an action), keyed by name. |
| `lucida_cohort` | A cohort, keyed by name. `kind` is fixed at create. |
| `lucida_flag` | A feature flag, keyed by its flag key. `kind` is fixed at create. |
| `lucida_insight` | A saved insight, keyed by name. |
| `lucida_metric` | A metric definition, keyed by name. |
| `lucida_experiment` | An experiment, keyed by name. Destroying archives it (the API has no delete). |
| `lucida_dashboard` | A dashboard with its tiles; tiles name their insight. |
| `lucida_alert` | An insight alert, keyed by name; `insight` names the insight it watches. |
| `lucida_key` | A project API key. The secret is returned once as the sensitive `secret` attribute; destroying revokes the key. |
| `lucida_export_schedule` | A batch export schedule. Where and what it exports is fixed at create; `credentials` are sent on create only. |
| `lucida_endpoint` | A query endpoint, keyed by slug; a change to query or variables creates a new version. |
| `lucida_connector` | A connector instance (source or destination), keyed by name. `credentials` are sent on create only. |
| `lucida_destination` | A destination connector (the same API object as `lucida_connector`; pick a destination `kind`). |
| `lucida_traffic_exclusion` | A traffic exclusion rule. The note is its identity; only `enabled` can change in place. |

## Notes

- Objects are matched by natural key (flag key, name, slug), never by id, exactly like the CLI. Creating an object whose key already exists fails; import it instead.
- Secrets: `lucida_key.secret` and credential attributes are `Sensitive` and live in Terraform state. Use an encrypted remote backend.
- Projects are created in the Lucida console (project creation needs a signed-in session, not an API key); use `data.lucida_project` to read one.
