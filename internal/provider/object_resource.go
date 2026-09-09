package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/withakedo/terraform-provider-i-doit/internal/client"
)

var (
	_ resource.Resource                = (*objectResource)(nil)
	_ resource.ResourceWithConfigure   = (*objectResource)(nil)
	_ resource.ResourceWithImportState = (*objectResource)(nil)
)

// NewObjectResource is the resource factory registered with the provider.
func NewObjectResource() resource.Resource { return &objectResource{} }

type objectResource struct {
	client *client.Client
}

type objectResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Type           types.String `tfsdk:"type"`
	Title          types.String `tfsdk:"title"`
	CmdbStatus     types.String `tfsdk:"cmdb_status"`
	TemplateID     types.Int64  `tfsdk:"template_id"`
	PurgeOnDestroy types.Bool   `tfsdk:"purge_on_destroy"`
	SysID          types.String `tfsdk:"sysid"`
	Status         types.String `tfsdk:"status"`
}

func (r *objectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_object"
}

func (r *objectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a CMDB object in i-doit. An object is identified by its object type constant and a title; category fields are managed separately with `idoit_category_entry`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Numeric i-doit object identifier.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Object type constant, e.g. `C__OBJTYPE__SERVER`. Changing this forces a new object.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "Object title.",
				Required:            true,
			},
			"cmdb_status": schema.StringAttribute{
				MarkdownDescription: "CMDB status to assign, as a constant (e.g. `C__CMDB_STATUS__IN_OPERATION`) or a numeric id. Written on create via `cmdb.object.create` and on update via the `" + catGlobal + "` category. Not drift-tracked; the resulting label is exposed in `status`.",
				Optional:            true,
			},
			"template_id": schema.Int64Attribute{
				MarkdownDescription: "Object id of a template object to clone when the object is created (`cmdb.object.create` `template` parameter). Changing this forces a new object.",
				Optional:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"purge_on_destroy": schema.BoolAttribute{
				MarkdownDescription: "When `true`, `terraform destroy` calls `cmdb.object.purge` (irreversible). When `false` (default) it calls `cmdb.object.archive`, which can be restored in i-doit.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"sysid": schema.StringAttribute{
				MarkdownDescription: "SYSID assigned by i-doit.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Current CMDB status label (e.g. `in operation`). Refreshed on every read.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *objectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data",
			fmt.Sprintf("Expected *client.Client, got %T. This is a provider bug.", req.ProviderData))
		return
	}
	r.client = c
}

func (r *objectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan objectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := r.client.CreateObjectFull(ctx,
		plan.Type.ValueString(),
		plan.Title.ValueString(),
		plan.CmdbStatus.ValueString(),
		plan.TemplateID.ValueInt64(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create i-doit object", err.Error())
		return
	}
	tflog.Info(ctx, "created i-doit object", map[string]any{"id": id, "type": plan.Type.ValueString()})

	plan.ID = types.StringValue(strconv.FormatInt(id, 10))

	obj, err := r.client.ReadObject(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read back created i-doit object", err.Error())
		return
	}
	if obj == nil {
		resp.Diagnostics.AddError("Created i-doit object not found",
			fmt.Sprintf("Object %d disappeared immediately after creation.", id))
		return
	}
	applyObjectRead(&plan, obj)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *objectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state objectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid object id in state", err.Error())
		return
	}

	obj, err := r.client.ReadObject(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read i-doit object", err.Error())
		return
	}
	if obj == nil {
		tflog.Warn(ctx, "i-doit object no longer exists, removing from state", map[string]any{"id": id})
		resp.State.RemoveResource(ctx)
		return
	}

	applyObjectRead(&state, obj)

	// Read is free to surface drift on the mutable title.
	if obj.Title != "" {
		state.Title = types.StringValue(obj.Title)
	}

	// On import the type is not known yet; adopt it from the API if it looks
	// like an object type constant.
	if state.Type.IsNull() || state.Type.ValueString() == "" {
		if strings.HasPrefix(obj.TypeConst, "C__OBJTYPE__") {
			state.Type = types.StringValue(obj.TypeConst)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *objectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state objectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid object id in state", err.Error())
		return
	}

	if plan.Title.ValueString() != state.Title.ValueString() {
		if err := r.client.UpdateObjectTitle(ctx, id, plan.Title.ValueString()); err != nil {
			resp.Diagnostics.AddError("Unable to update i-doit object", err.Error())
			return
		}
	}

	if !plan.CmdbStatus.IsNull() && plan.CmdbStatus.ValueString() != state.CmdbStatus.ValueString() {
		entryID := int64(0)
		if e, _ := firstCategoryEntry(ctx, r.client, id, catGlobal); e != nil {
			entryID = e.EntryID()
		}
		if _, err := r.client.SaveCategory(ctx, id, catGlobal, map[string]any{"cmdb_status": plan.CmdbStatus.ValueString()}, entryID); err != nil {
			resp.Diagnostics.AddError("Unable to update CMDB status", err.Error())
			return
		}
	}

	plan.ID = state.ID
	obj, err := r.client.ReadObject(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read back updated i-doit object", err.Error())
		return
	}
	if obj == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	applyObjectRead(&plan, obj)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *objectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state objectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid object id in state", err.Error())
		return
	}

	if state.PurgeOnDestroy.ValueBool() {
		if err := r.client.PurgeObject(ctx, id); err != nil {
			resp.Diagnostics.AddError("Unable to purge i-doit object", err.Error())
		}
		return
	}
	if err := r.client.ArchiveObject(ctx, id); err != nil {
		resp.Diagnostics.AddError("Unable to archive i-doit object", err.Error())
	}
}

func (r *objectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// applyObjectRead copies the computed attributes from an API object into the
// model. It deliberately leaves the configured attributes (type, title) alone so
// Create and Update never produce a result that differs from the plan.
func applyObjectRead(m *objectResourceModel, obj *client.Object) {
	m.SysID = types.StringValue(obj.SysID)
	m.Status = types.StringValue(obj.StatusLabel())
	if m.PurgeOnDestroy.IsNull() || m.PurgeOnDestroy.IsUnknown() {
		m.PurgeOnDestroy = types.BoolValue(false)
	}
}
