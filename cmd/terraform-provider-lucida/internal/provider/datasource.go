package provider

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"lucida/internal/defsync"
	"lucida/internal/defsync/apiremote"
)

type projectData struct{ c *Client }

func newProjectData() datasource.DataSource { return &projectData{} }

func (d *projectData) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (d *projectData) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a project; defaults to the provider's project.",
		Attributes: map[string]schema.Attribute{
			"id":   schema.StringAttribute{Optional: true, Computed: true, Description: "Project id (default: the provider's project_id)."},
			"name": schema.StringAttribute{Computed: true},
		},
	}
}

func (d *projectData) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	d.c, _ = req.ProviderData.(*Client)
}

func (d *projectData) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var m struct {
		ID   types.String `tfsdk:"id"`
		Name types.String `tfsdk:"name"`
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := m.ID.ValueString()
	if id == "" {
		id = d.c.ProjectID
	}
	var out struct{ ID, Name string }
	if err := d.c.Root(ctx, "GET", "/v1/projects/"+url.PathEscape(id), nil, nil, &out); err != nil {
		resp.Diagnostics.AddError("Reading project failed", err.Error())
		return
	}
	m.ID, m.Name = types.StringValue(id), types.StringValue(out.Name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

type flagData struct{ c *Client }

func newFlagData() datasource.DataSource { return &flagData{} }

func (d *flagData) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flag"
}

func (d *flagData) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a feature flag by key, for wiring ids into other resources.",
		Attributes: map[string]schema.Attribute{
			"key":        schema.StringAttribute{Required: true},
			"id":         schema.StringAttribute{Computed: true},
			"name":       schema.StringAttribute{Computed: true},
			"enabled":    schema.BoolAttribute{Computed: true},
			"definition": schema.StringAttribute{Computed: true, Description: "JSON of the flag's managed fields, as the resource would hold it."},
		},
	}
}

func (d *flagData) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	d.c, _ = req.ProviderData.(*Client)
}

func (d *flagData) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var m struct {
		Key        types.String `tfsdk:"key"`
		ID         types.String `tfsdk:"id"`
		Name       types.String `tfsdk:"name"`
		Enabled    types.Bool   `tfsdk:"enabled"`
		Definition types.String `tfsdk:"definition"`
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	k, _ := defsync.KindByName("flags")
	rows, err := apiremote.New(d.c, nil).List(ctx, k)
	if err != nil {
		resp.Diagnostics.AddError("Listing flags failed", err.Error())
		return
	}
	for _, o := range rows {
		if o["key"] != m.Key.ValueString() {
			continue
		}
		n := defsync.Normalize(k, o)
		delete(n, "key")
		m.ID = types.StringValue(fmt.Sprint(o["id"]))
		nm, _ := n["name"].(string)
		m.Name = types.StringValue(nm)
		en, _ := n["enabled"].(bool)
		m.Enabled, m.Definition = types.BoolValue(en), types.StringValue(canonical(n))
		resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
		return
	}
	resp.Diagnostics.AddError("Flag not found", fmt.Sprintf("no flag with key %q in project %s", m.Key.ValueString(), d.c.ProjectID))
}
