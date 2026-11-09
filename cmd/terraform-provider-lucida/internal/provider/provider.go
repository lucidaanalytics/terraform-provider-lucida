// Package provider is the Terraform provider for Lucida. Resources come from Specs(), which is
// derived from lucida/internal/defsync, so a new definitions kind becomes a resource automatically.
package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DefaultHost is the hosted Lucida.
const DefaultHost = "https://lucidace.com"

type lucida struct{ version string }

// New returns the provider factory.
func New(version string) func() provider.Provider {
	return func() provider.Provider { return &lucida{version: version} }
}

type config struct {
	Host      types.String `tfsdk:"host"`
	APIKey    types.String `tfsdk:"api_key"`
	ProjectID types.String `tfsdk:"project_id"`
}

func (p *lucida) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName, resp.Version = "lucida", p.version
}

func (p *lucida) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage Lucida project definitions as code. The resources are the same kinds `lucida pull|plan|apply` manages.",
		Attributes: map[string]schema.Attribute{
			"host":       schema.StringAttribute{Optional: true, Description: "Lucida API origin. Defaults to $LUCIDA_HOST or " + DefaultHost + "."},
			"api_key":    schema.StringAttribute{Optional: true, Sensitive: true, Description: "API key with the scopes the resources need. Defaults to $LUCIDA_API_KEY."},
			"project_id": schema.StringAttribute{Optional: true, Description: "Project every resource lives in. Defaults to $LUCIDA_PROJECT_ID."},
		},
	}
}

func pick(v types.String, env, def string) string {
	if !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
		return v.ValueString()
	}
	if s := os.Getenv(env); s != "" {
		return s
	}
	return def
}

func (p *lucida) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var c config
	resp.Diagnostics.Append(req.Config.Get(ctx, &c)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cl := &Client{Host: pick(c.Host, "LUCIDA_HOST", DefaultHost), APIKey: pick(c.APIKey, "LUCIDA_API_KEY", ""), ProjectID: pick(c.ProjectID, "LUCIDA_PROJECT_ID", "")}
	if cl.APIKey == "" {
		resp.Diagnostics.AddAttributeError(path.Root("api_key"), "Missing API key", "Set api_key or the LUCIDA_API_KEY environment variable.")
	}
	if cl.ProjectID == "" {
		resp.Diagnostics.AddAttributeError(path.Root("project_id"), "Missing project", "Set project_id or the LUCIDA_PROJECT_ID environment variable.")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.ResourceData, resp.DataSourceData = cl, cl
}

func (p *lucida) Resources(context.Context) []func() resource.Resource {
	var out []func() resource.Resource
	for _, s := range Specs() {
		out = append(out, newResource(s))
	}
	return out
}

func (p *lucida) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{newProjectData, newFlagData}
}
