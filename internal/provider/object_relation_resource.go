package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/withakedo/terraform-provider-i-doit/internal/client"
)

var (
	_ resource.Resource                = (*objectRelationResource)(nil)
	_ resource.ResourceWithConfigure   = (*objectRelationResource)(nil)
	_ resource.ResourceWithImportState = (*objectRelationResource)(nil)
)

// NewObjectRelationResource is the resource factory registered with the provider.
func NewObjectRelationResource() resource.Resource { return &objectRelationResource{} }

type objectRelationResource struct {
	client *client.Client
}

type objectRelationResourceModel struct {
	ID             types.String `tfsdk:"id"`
	MasterObjectID types.String `tfsdk:"master_object_id"`
	SlaveObjectID  types.String `tfsdk:"slave_object_id"`
	RelationType   types.String `tfsdk:"relation_type"`
	Description    types.String `tfsdk:"description"`
	Weighting      types.String `tfsdk:"weighting"`
	Extra          types.Map    `tfsdk:"extra"`
	EntryID        types.Int64  `tfsdk:"entry_id"`
}

func (m objectRelationResourceModel) masterID() (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(m.MasterObjectID.ValueString()), 10, 64)
}

func (r *objectRelationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_object_relation"
}

func (r *objectRelationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a relation between two CMDB objects via the `" + catRelation + "` category on the master object.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "`<master_object_id>/<entry_id>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"master_object_id": schema.StringAttribute{
				MarkdownDescription: "Object id on the master side of the relation. Changing this forces a new relation.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"slave_object_id": schema.StringAttribute{
				MarkdownDescription: "Object id on the slave side of the relation. Changing this forces a new relation.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"relation_type": schema.StringAttribute{
				MarkdownDescription: "Relation type, as a constant (e.g. `C__RELATION_TYPE__SOFTWARE`) or a numeric id. Changing this forces a new relation.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": optionalComputedString("Free-text description of the relation."),
			"weighting":   optionalComputedString("Optional weighting value stored on the relation."),
			"extra": schema.MapAttribute{
				MarkdownDescription: "Additional `" + catRelation + "` fields passed verbatim to `cmdb.category.save`. Not drift-tracked.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"entry_id": schema.Int64Attribute{
				MarkdownDescription: "Numeric id of the relation category entry.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *objectRelationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *objectRelationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan objectRelationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	master, err := plan.masterID()
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("master_object_id"), "Invalid master_object_id", err.Error())
		return
	}

	data, diags := r.buildData(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	entryID, err := r.client.SaveCategory(ctx, master, catRelation, data, 0)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create relation", err.Error())
		return
	}
	tflog.Info(ctx, "created object relation", map[string]any{"master": master, "entry": entryID})

	plan.EntryID = types.Int64Value(entryID)
	plan.ID = types.StringValue(fmt.Sprintf("%d/%d", master, entryID))
	if err := r.refresh(ctx, &plan, master, entryID, false); err != nil {
		resp.Diagnostics.AddError("Unable to read back relation", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *objectRelationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state objectRelationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	master, err := state.masterID()
	if err != nil {
		resp.Diagnostics.AddError("Invalid master_object_id in state", err.Error())
		return
	}
	entryID := state.EntryID.ValueInt64()

	entry, err := findCategoryEntry(ctx, r.client, master, catRelation, entryID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read relation", err.Error())
		return
	}
	if entry == nil {
		tflog.Warn(ctx, "object relation gone, removing from state", map[string]any{"master": master, "entry": entryID})
		resp.State.RemoveResource(ctx)
		return
	}
	if err := r.refresh(ctx, &state, master, entryID, true); err != nil {
		resp.Diagnostics.AddError("Unable to read relation", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *objectRelationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state objectRelationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	master, err := state.masterID()
	if err != nil {
		resp.Diagnostics.AddError("Invalid master_object_id in state", err.Error())
		return
	}
	entryID := state.EntryID.ValueInt64()
	plan.ID = state.ID
	plan.EntryID = state.EntryID

	data, diags := r.buildData(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.SaveCategory(ctx, master, catRelation, data, entryID); err != nil {
		resp.Diagnostics.AddError("Unable to update relation", err.Error())
		return
	}
	if err := r.refresh(ctx, &plan, master, entryID, false); err != nil {
		resp.Diagnostics.AddError("Unable to read back relation", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *objectRelationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state objectRelationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	master, err := state.masterID()
	if err != nil {
		resp.Diagnostics.AddError("Invalid master_object_id in state", err.Error())
		return
	}
	if err := r.client.DeleteCategoryEntry(ctx, master, catRelation, state.EntryID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Unable to delete relation", err.Error())
	}
}

func (r *objectRelationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import id",
			"Expected `<master_object_id>/<entry_id>`, got "+req.ID)
		return
	}
	entryID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import id", "entry_id must be numeric: "+err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("master_object_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("entry_id"), types.Int64Value(entryID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *objectRelationResource) buildData(ctx context.Context, m *objectRelationResourceModel) (map[string]any, diag.Diagnostics) {
	data := map[string]any{
		"master":        m.MasterObjectID.ValueString(),
		"slave":         m.SlaveObjectID.ValueString(),
		"relation_type": m.RelationType.ValueString(),
	}
	putStr(data, "description", m.Description)
	putStr(data, "weighting", m.Weighting)

	var diags diag.Diagnostics
	diags.Append(mergeExtra(ctx, data, m.Extra)...)
	return data, diags
}

func (r *objectRelationResource) refresh(ctx context.Context, m *objectRelationResourceModel, master, entryID int64, fromRead bool) error {
	entry, err := findCategoryEntry(ctx, r.client, master, catRelation, entryID)
	if err != nil {
		return err
	}
	if entry != nil {
		setStrFromEntry(&m.Description, entry, "description")
		setStrFromEntry(&m.Weighting, entry, "weighting")
		if fromRead {
			setRefFromEntry(&m.MasterObjectID, entry, "master")
			setRefFromEntry(&m.SlaveObjectID, entry, "slave")
			if rt := relationTypeValue(entry); rt != "" {
				m.RelationType = types.StringValue(rt)
			}
		}
	}
	nullUnknownStr(&m.Description, &m.Weighting)
	return nil
}

// relationTypeValue extracts the relation type constant (preferred) or title
// from a C__CATG__RELATION entry.
func relationTypeValue(entry map[string]any) string {
	raw, ok := entry["relation_type"]
	if !ok {
		return ""
	}
	if mm, ok := raw.(map[string]any); ok {
		for _, k := range []string{"const", "constant", "title"} {
			if s, ok := coerceScalar(mm[k]); ok && s != "" {
				return s
			}
		}
		return ""
	}
	if s, ok := coerceScalar(raw); ok {
		return s
	}
	return ""
}
