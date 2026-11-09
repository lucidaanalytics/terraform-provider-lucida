package provider

import (
	"fmt"
	"slices"
	"strings"
)

// examples is the HCL for each resource: the key value, then the jsonencode body of `definition`.
// A new Spec needs a row here (TestEveryResourceHasAnExample), which keeps the docs honest.
var examples = map[string]struct{ Key, Def string }{
	"action":            {"signed_up", `{ description = "A visitor created an account", steps = [{ event = "signed_up" }] }`},
	"cohort":            {"power_users", `{ kind = "dynamic", filters = { match = "all", conditions = [{ event = "report_viewed", op = "gte", count = 5, days = 30 }] } }`},
	"flag":              {"new-checkout", `{ name = "New checkout", enabled = true, filters = { rollout = 25 } }`},
	"insight":           {"Weekly signups", `{ description = "Signups per week", query = { kind = "trend", series = [{ event = "signed_up" }], interval = "week" } }`},
	"metric":            {"checkout_conversion", `{ display_name = "Checkout conversion", unit = "ratio", kind = "funnel", definition = { steps = ["cart_viewed", "order_completed"] } }`},
	"experiment":        {"checkout-button-colour", `{ hypothesis = "A green button lifts checkout starts", kind = "feature", variants = [{ key = "control" }, { key = "green" }], traffic_pct = 50 }`},
	"dashboard":         {"Growth", `{ description = "Weekly growth", tiles = [{ insight = "Weekly signups", position = 0 }] }`},
	"alert":             {"Signups drop", `{ insight = "Weekly signups", condition = { kind = "threshold", op = "lt", value = 100 }, check_interval = "1h", enabled = true }`},
	"key":               {"ci-read", `{ kind = "project", scopes = ["read"] }`},
	"export_schedule":   {"daily-events", `{ stream = "events", format = "parquet", destination = "s3", interval_secs = 86400 }`},
	"endpoint":          {"top-pages", `{ name = "Top pages", refresh_every_minutes = 60, query = { kind = "trend", series = [{ event = "$pageview" }] } }`},
	"connector":         {"warehouse-sync", `{ kind = "postgres", config = { host = "db.internal", database = "app" }, sync_interval_secs = 900 }`},
	"destination":       {"slack-alerts", `{ kind = "slack", config = { channel = "#growth" } }`},
	"traffic_exclusion": {"Office network", `{ kind = "ip", value = "203.0.113.0/24", enabled = true }`},
}

// refExample is the sensitive attribute line for kinds that take credential references.
const refExample = "  credentials = jsonencode({ token = var.lucida_token }) # sensitive; sent on create only\n"

// ResourceExample is the resource.tf for a spec.
func ResourceExample(s Spec) string {
	ex := examples[s.TypeSuffix]
	var b strings.Builder
	fmt.Fprintf(&b, "resource \"lucida_%s\" \"example\" {\n  %s = %q\n  definition = jsonencode(%s)\n", s.TypeSuffix, s.Kind.Key, ex.Key, ex.Def)
	if len(s.Kind.Refs) > 0 {
		b.WriteString(refExample)
	}
	b.WriteString("}\n")
	return b.String()
}

// ProviderExample is examples/provider/provider.tf.
const ProviderExample = `terraform {
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
`

func front(title, desc string) string {
	return fmt.Sprintf("---\npage_title: %q\nsubcategory: \"\"\ndescription: |-\n  %s\n---\n\n", title, desc)
}

// ResourceDoc renders docs/resources/<suffix>.md.
func ResourceDoc(s Spec) string {
	k := s.Kind
	name := "lucida_" + s.TypeSuffix
	var b strings.Builder
	b.WriteString(front(name+" (Resource)", s.Summary))
	fmt.Fprintf(&b, "# %s (Resource)\n\n%s\n\nManages the `%s` collection, the same kind `lucida pull|plan|apply` calls `%s`.\n\n## Example Usage\n\n```terraform\n%s```\n\n", name, s.Summary, k.Name, k.Name, ResourceExample(s))
	b.WriteString("## Schema\n\n### Required\n\n")
	fmt.Fprintf(&b, "- `%s` (String) Natural key. Changing it replaces the resource.\n", k.Key)
	fmt.Fprintf(&b, "- `definition` (String) JSON object of the managed fields: `%s`.%s\n", strings.Join(definable(k), "`, `"), fixedNote(s))
	if len(k.Refs) > 0 || s.Secret {
		b.WriteString("\n### Optional\n\n")
		for _, r := range k.Refs {
			fmt.Fprintf(&b, "- `%s` (String, Sensitive) JSON object, sent on create only and never read back. Rotate it with `terraform apply -replace`.\n", r)
		}
	}
	b.WriteString("\n### Read-Only\n\n- `id` (String) The Lucida id (the slug for endpoints).\n")
	if s.Secret {
		b.WriteString("- `secret` (String, Sensitive) The key secret, returned once at create. It is stored in Terraform state: protect the state.\n")
	}
	b.WriteString("\n## Drift\n\nOn every refresh the fields named in `definition` are compared with Lucida; a change made in the UI shows as a diff. Fields the definition leaves out are never compared.\n")
	fmt.Fprintf(&b, "\n## Import\n\nImport takes the Lucida id (the slug for endpoints):\n\n```shell\nterraform import %s.example <id>\n```\n", name)
	return b.String()
}

// IndexDoc renders docs/index.md.
func IndexDoc() string {
	var b strings.Builder
	b.WriteString(front("Lucida Provider", "Manage Lucida project definitions as code."))
	b.WriteString("# Lucida Provider\n\nManages a Lucida project's definitions. The resources are generated from the same model as the `lucida` CLI (`lucida pull`, `plan`, `apply`), so Terraform and the CLI see identical kinds and fields.\n\n")
	b.WriteString("## Example Usage\n\n```terraform\n" + ProviderExample + "```\n\n## Schema\n\n### Optional\n\n")
	b.WriteString("- `host` (String) Lucida API origin. Defaults to `$LUCIDA_HOST` or `" + DefaultHost + "`.\n")
	b.WriteString("- `api_key` (String, Sensitive) API key. Defaults to `$LUCIDA_API_KEY`. Use a key scoped to what you manage.\n")
	b.WriteString("- `project_id` (String) Project every resource lives in. Defaults to `$LUCIDA_PROJECT_ID`.\n\n")
	b.WriteString("## Resources\n\n| Resource | Manages |\n| --- | --- |\n")
	for _, s := range Specs() {
		fmt.Fprintf(&b, "| `lucida_%s` | %s |\n", s.TypeSuffix, s.Summary)
	}
	b.WriteString("\n## Notes\n\n- Objects are matched by natural key (flag key, name, slug), never by id, exactly like the CLI. Creating an object whose key already exists fails; import it instead.\n")
	b.WriteString("- Secrets: `lucida_key.secret` and credential attributes are `Sensitive` and live in Terraform state. Use an encrypted remote backend.\n")
	b.WriteString("- Projects are created in the Lucida console (project creation needs a signed-in session, not an API key); use `data.lucida_project` to read one.\n")
	return b.String()
}

// DataDocs returns docs/data-sources/<name>.md contents by file name.
func DataDocs() map[string]string {
	return map[string]string{
		"project.md": front("lucida_project (Data Source)", "Looks up a project.") + "# lucida_project (Data Source)\n\nLooks up a project; defaults to the provider's project.\n\n## Example Usage\n\n```terraform\ndata \"lucida_project\" \"this\" {}\n```\n\n## Schema\n\n### Optional\n\n- `id` (String) Project id (default: the provider's `project_id`).\n\n### Read-Only\n\n- `name` (String)\n",
		"flag.md":    front("lucida_flag (Data Source)", "Looks up a feature flag by key.") + "# lucida_flag (Data Source)\n\nLooks up a feature flag by key, for wiring ids into other resources.\n\n## Example Usage\n\n```terraform\ndata \"lucida_flag\" \"checkout\" {\n  key = \"new-checkout\"\n}\n```\n\n## Schema\n\n### Required\n\n- `key` (String)\n\n### Read-Only\n\n- `id` (String)\n- `name` (String)\n- `enabled` (Boolean)\n- `definition` (String) JSON of the flag's managed fields, as the resource would hold it.\n",
	}
}

// Files returns every generated file (path relative to the provider directory) and its content.
func Files() map[string]string {
	out := map[string]string{
		"docs/index.md":                                       IndexDoc(),
		"examples/provider/provider.tf":                       ProviderExample,
		"examples/data-sources/lucida_project/data-source.tf": "data \"lucida_project\" \"this\" {}\n",
		"examples/data-sources/lucida_flag/data-source.tf":    "data \"lucida_flag\" \"checkout\" {\n  key = \"new-checkout\"\n}\n",
	}
	for n, c := range DataDocs() {
		out["docs/data-sources/"+n] = c
	}
	for _, s := range Specs() {
		out["docs/resources/"+s.TypeSuffix+".md"] = ResourceDoc(s)
		out["examples/resources/lucida_"+s.TypeSuffix+"/resource.tf"] = ResourceExample(s)
		out["examples/resources/lucida_"+s.TypeSuffix+"/import.sh"] = fmt.Sprintf("terraform import lucida_%s.example <id>\n", s.TypeSuffix)
	}
	return out
}

// sortedKeys is a small helper for deterministic iteration.
func sortedKeys[V any](m map[string]V) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	slices.Sort(ks)
	return ks
}
