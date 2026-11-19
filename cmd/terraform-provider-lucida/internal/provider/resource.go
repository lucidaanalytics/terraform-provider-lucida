package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// attrs reads and writes attributes of a plan, config or state.
type attrs interface {
	GetAttribute(ctx context.Context, p path.Path, target any) diag.Diagnostics
}

type attrSetter interface {
	SetAttribute(ctx context.Context, p path.Path, val any) diag.Diagnostics
}

// kindResource is the one generic resource behind every Spec.
type kindResource struct {
	spec Spec
	crud *Crud
}

func newResource(s Spec) func() resource.Resource {
	return func() resource.Resource { return &kindResource{spec: s} }
}

var (
	_ resource.ResourceWithImportState    = (*kindResource)(nil)
	_ resource.ResourceWithValidateConfig = (*kindResource)(nil)
	_ resource.ResourceWithModifyPlan     = (*kindResource)(nil)
	_ resource.ResourceWithConfigure      = (*kindResource)(nil)
)

func (r *kindResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.spec.TypeSuffix
}

func (r *kindResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	k := r.spec.Kind
	a := map[string]schema.Attribute{
		attrID: schema.StringAttribute{Computed: true, Description: "The Lucida id (the slug for endpoints).",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		k.Key: r.keyAttribute(),
		attrDefinition: schema.StringAttribute{Required: true, MarkdownDescription: "JSON object (`jsonencode({...})`) of the managed fields: " +
			fieldList(k) + "." + fixedNote(r.spec)},
	}
	if r.spec.Secret {
		a[attrSecret] = schema.StringAttribute{Computed: true, Sensitive: true, Description: "The key secret, returned once at create.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}
	}
	for _, ref := range k.Refs {
		a[ref] = schema.StringAttribute{Optional: true, Sensitive: true, MarkdownDescription: "Sensitive JSON object (`jsonencode({...})`), sent on create only and never read back. Rotate it with `terraform apply -replace`."}
	}
	resp.Schema = schema.Schema{Description: r.spec.Summary, Attributes: a}
}

func (r *kindResource) keyAttribute() schema.Attribute {
	if r.spec.KeyValue != "" {
		return schema.StringAttribute{Computed: true, Description: "Fixed key of this singleton (`" + r.spec.KeyValue + "`).",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}
	}
	return schema.StringAttribute{Required: true, Description: "Natural key. Changing it replaces the resource.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}}
}

func fixedNote(s Spec) string {
	fixed := append(append([]string{}, s.Kind.Fixed...), s.Kind.Frozen...)
	switch {
	case s.Kind.Once:
		return " Created once: any change replaces the resource."
	case len(fixed) > 0:
		return " Changing `" + strings.Join(fixed, "`, `") + "` replaces the resource."
	}
	return ""
}

func (r *kindResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if c, ok := req.ProviderData.(*Client); ok {
		r.crud = &Crud{Spec: r.spec, API: c}
	}
}

func (r *kindResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var def types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(attrDefinition), &def)...)
	if !def.IsNull() && !def.IsUnknown() {
		if err := validateDefinition(r.spec.Kind, def.ValueString()); err != nil {
			resp.Diagnostics.AddAttributeError(path.Root(attrDefinition), "Invalid definition", err.Error())
		}
	}
	for _, ref := range r.spec.Kind.Refs {
		var v types.String
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(ref), &v)...)
		if !v.IsNull() && !v.IsUnknown() {
			if _, err := parseObject(v.ValueString()); err != nil {
				resp.Diagnostics.AddAttributeError(path.Root(ref), "Invalid "+ref, err.Error()) // never echoes the value
			}
		}
	}
}

func (r *kindResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var old, planned types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root(attrDefinition), &old)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root(attrDefinition), &planned)...)
	if resp.Diagnostics.HasError() || old.IsNull() || planned.IsNull() || planned.IsUnknown() {
		return
	}
	o, err1 := parseObject(old.ValueString())
	p, err2 := parseObject(planned.ValueString())
	if err1 == nil && err2 == nil && needsReplace(r.spec.Kind, o, p) {
		resp.RequiresReplace = append(resp.RequiresReplace, path.Root(attrDefinition))
	}
}

// entity builds an Entity from a plan or state.
func (r *kindResource) entity(ctx context.Context, g attrs) (Entity, diag.Diagnostics) {
	var d diag.Diagnostics
	var id, key, def, secret types.String
	d.Append(g.GetAttribute(ctx, path.Root(attrID), &id)...)
	d.Append(g.GetAttribute(ctx, path.Root(r.spec.Kind.Key), &key)...)
	d.Append(g.GetAttribute(ctx, path.Root(attrDefinition), &def)...)
	e := Entity{ID: id.ValueString(), Key: key.ValueString()}
	if r.spec.KeyValue != "" {
		e.Key = r.spec.KeyValue
	}
	if r.spec.Secret {
		d.Append(g.GetAttribute(ctx, path.Root(attrSecret), &secret)...)
		e.Secret = secret.ValueString()
	}
	if !def.IsNull() {
		var err error
		if e.Def, err = parseObject(def.ValueString()); err != nil {
			d.AddAttributeError(path.Root(attrDefinition), "Invalid definition", err.Error())
		}
	}
	for _, ref := range r.spec.Kind.Refs {
		var v types.String
		d.Append(g.GetAttribute(ctx, path.Root(ref), &v)...)
		if v.IsNull() || v.IsUnknown() {
			continue
		}
		o, err := parseObject(v.ValueString())
		if err != nil {
			d.AddAttributeError(path.Root(ref), "Invalid "+ref, err.Error())
			continue
		}
		if e.Refs == nil {
			e.Refs = map[string]any{}
		}
		e.Refs[ref] = o
	}
	return e, d
}

// store writes the server-derived attributes. def is the exact string to keep in state.
func (r *kindResource) store(ctx context.Context, s attrSetter, e Entity, def string) diag.Diagnostics {
	var d diag.Diagnostics
	d.Append(s.SetAttribute(ctx, path.Root(attrID), e.ID)...)
	d.Append(s.SetAttribute(ctx, path.Root(r.spec.Kind.Key), e.Key)...)
	d.Append(s.SetAttribute(ctx, path.Root(attrDefinition), def)...)
	if r.spec.Secret {
		d.Append(s.SetAttribute(ctx, path.Root(attrSecret), e.Secret)...)
	}
	return d
}

func (r *kindResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	e, d := r.entity(ctx, req.Plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	var def types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root(attrDefinition), &def)...)
	created, err := r.crud.Create(ctx, e)
	if err != nil {
		resp.Diagnostics.AddError("Creating "+r.spec.TypeSuffix+" failed", err.Error())
		return
	}
	resp.State.Raw = req.Plan.Raw
	resp.Diagnostics.Append(r.store(ctx, &resp.State, created, def.ValueString())...)
}

func (r *kindResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var id, def types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root(attrID), &id)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root(attrDefinition), &def)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var prior *Entity
	if !def.IsNull() { // null after ImportState: report everything
		p, d := r.entity(ctx, req.State)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		prior = &p
	}
	got, found, err := r.crud.Read(ctx, id.ValueString(), prior)
	if err != nil {
		resp.Diagnostics.AddError("Reading "+r.spec.TypeSuffix+" failed", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	text := canonical(got.Def)
	if prior != nil && sameJSON(def.ValueString(), text) {
		text = def.ValueString() // no drift: keep the user's spelling
	}
	resp.Diagnostics.Append(r.store(ctx, &resp.State, got, text)...)
}

func (r *kindResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	e, d := r.entity(ctx, req.Plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	var def, id types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root(attrDefinition), &def)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root(attrID), &id)...)
	e.ID = id.ValueString()
	if err := r.crud.Update(ctx, e); err != nil {
		resp.Diagnostics.AddError("Updating "+r.spec.TypeSuffix+" failed", err.Error())
		return
	}
	resp.State.Raw = req.Plan.Raw
	resp.Diagnostics.Append(r.store(ctx, &resp.State, e, def.ValueString())...)
}

func (r *kindResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var id types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root(attrID), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.crud.Delete(ctx, id.ValueString()); err != nil {
		resp.Diagnostics.AddError("Deleting "+r.spec.TypeSuffix+" failed", fmt.Sprint(err))
	}
}

// ImportState takes the Lucida id (the slug for endpoints); Read then fills in the rest.
func (r *kindResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root(attrID), req, resp)
}
