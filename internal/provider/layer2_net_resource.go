package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/withakedo/terraform-provider-i-doit/internal/client"
)

var (
	_ resource.Resource                = (*layer2NetResource)(nil)
	_ resource.ResourceWithConfigure   = (*layer2NetResource)(nil)
	_ resource.ResourceWithImportState = (*layer2NetResource)(nil)
)

// NewLayer2NetResource is the resource factory registered with the provider.
func NewLayer2NetResource() resource.Resource { return &layer2NetResource{} }

type layer2NetResource struct {
	client *client.Client
}

type layer2NetResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Title          types.String `tfsdk:"title"`
	VLANID         types.String `tfsdk:"vlan_id"`
	Description    types.String `tfsdk:"description"`
	Standard       types.Bool   `tfsdk:"standard"`
	Layer3NetIDs   types.List   `tfsdk:"layer3_net_ids"`
	Extra          types.Map    `tfsdk:"extra"`
	PurgeOnDestroy types.Bool   `tfsdk:"purge_on_destroy"`
	SysID          types.String `tfsdk:"sysid"`
}

func (r *layer2NetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_layer2_net"
}

func (r *layer2NetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Layer-2 network (VLAN) in i-doit: a `" + objTypeLayer2Net + "` object together with its `" + catL2Net + "` category.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Numeric i-doit object id of the Layer-2 net.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "Object title, e.g. `VLAN 100 - servers`.",
				Required:            true,
			},
			"vlan_id":     optionalComputedString("VLAN id, e.g. `100`."),
			"description": optionalComputedString("Free-text description stored on the Layer-2 category."),
			"standard": schema.BoolAttribute{
				MarkdownDescription: "Mark this VLAN as the standard / native VLAN.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"layer3_net_ids": schema.ListAttribute{
				MarkdownDescription: "Object ids of the Layer-3 nets this VLAN is assigned to (written to `" + catL2Net + "` as `assigned_nets`). Not drift-tracked; if your i-doit version uses a different key, set it through `extra`.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"extra": schema.MapAttribute{
				MarkdownDescription: "Additional `" + catL2Net + "` fields passed verbatim to `cmdb.category.save`. Not drift-tracked.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"purge_on_destroy": schema.BoolAttribute{
				MarkdownDescription: "When `true`, `terraform destroy` purges the net object irreversibly instead of archiving it.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"sysid": schema.StringAttribute{
				MarkdownDescription: "SYSID assigned by i-doit.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *layer2NetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *layer2NetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan layer2NetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := r.client.CreateObject(ctx, objTypeLayer2Net, plan.Title.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Layer-2 net", err.Error())
		return
	}
	plan.ID = types.StringValue(strconv.FormatInt(id, 10))
	tflog.Info(ctx, "created Layer-2 net", map[string]any{"id": id})

	data, diags := r.buildData(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.SaveCategory(ctx, id, catL2Net, data, 0); err != nil {
		resp.Diagnostics.AddError("Unable to write Layer-2 category", err.Error())
		return
	}

	if err := r.refresh(ctx, &plan, id, false); err != nil {
		resp.Diagnostics.AddError("Unable to read back Layer-2 net", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *layer2NetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state layer2NetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid id in state", err.Error())
		return
	}

	obj, err := r.client.ReadObject(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Layer-2 net", err.Error())
		return
	}
	if obj == nil {
		tflog.Warn(ctx, "Layer-2 net gone, removing from state", map[string]any{"id": id})
		resp.State.RemoveResource(ctx)
		return
	}
	if err := r.refresh(ctx, &state, id, true); err != nil {
		resp.Diagnostics.AddError("Unable to read Layer-2 category", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *layer2NetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state layer2NetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid id in state", err.Error())
		return
	}
	plan.ID = state.ID

	if plan.Title.ValueString() != state.Title.ValueString() {
		if err := r.client.UpdateObjectTitle(ctx, id, plan.Title.ValueString()); err != nil {
			resp.Diagnostics.AddError("Unable to update net title", err.Error())
			return
		}
	}

	data, diags := r.buildData(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	entryID := int64(0)
	if e, _ := firstCategoryEntry(ctx, r.client, id, catL2Net); e != nil {
		entryID = e.EntryID()
	}
	if _, err := r.client.SaveCategory(ctx, id, catL2Net, data, entryID); err != nil {
		resp.Diagnostics.AddError("Unable to update Layer-2 category", err.Error())
		return
	}

	if err := r.refresh(ctx, &plan, id, false); err != nil {
		resp.Diagnostics.AddError("Unable to read back Layer-2 net", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *layer2NetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state layer2NetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid id in state", err.Error())
		return
	}
	deleteObject(ctx, r.client, id, state.PurgeOnDestroy.ValueBool(), &resp.Diagnostics)
}

func (r *layer2NetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *layer2NetResource) buildData(ctx context.Context, m *layer2NetResourceModel) (map[string]any, diag.Diagnostics) {
	data := map[string]any{}
	putStr(data, "vlan_id", m.VLANID)
	putStr(data, "description", m.Description)
	putBool(data, "standard", m.Standard)

	var diags diag.Diagnostics
	diags.Append(putStrList(ctx, data, "assigned_nets", m.Layer3NetIDs)...)
	diags.Append(mergeExtra(ctx, data, m.Extra)...)
	return data, diags
}

func (r *layer2NetResource) refresh(ctx context.Context, m *layer2NetResourceModel, id int64, fromRead bool) error {
	obj, err := r.client.ReadObject(ctx, id)
	if err != nil {
		return err
	}
	if obj != nil {
		if obj.Title != "" && (fromRead || m.Title.IsNull() || m.Title.IsUnknown()) {
			m.Title = types.StringValue(obj.Title)
		}
		m.SysID = types.StringValue(obj.SysID)
	}
	if m.PurgeOnDestroy.IsNull() || m.PurgeOnDestroy.IsUnknown() {
		m.PurgeOnDestroy = types.BoolValue(false)
	}

	e, err := firstCategoryEntry(ctx, r.client, id, catL2Net)
	if err != nil {
		return err
	}
	if e != nil {
		setStrFromEntry(&m.VLANID, e, "vlan_id")
		setStrFromEntry(&m.Description, e, "description")
		setBoolFromEntry(&m.Standard, e, "standard")
	}
	nullUnknownStr(&m.SysID, &m.VLANID, &m.Description)
	nullUnknownBool(&m.Standard)
	return nil
}
