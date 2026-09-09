package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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
	_ resource.Resource                = (*dialogValueResource)(nil)
	_ resource.ResourceWithConfigure   = (*dialogValueResource)(nil)
	_ resource.ResourceWithImportState = (*dialogValueResource)(nil)
)

// NewDialogValueResource is the resource factory registered with the provider.
func NewDialogValueResource() resource.Resource { return &dialogValueResource{} }

type dialogValueResource struct {
	client *client.Client
}

type dialogValueResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Category types.String `tfsdk:"category"`
	Property types.String `tfsdk:"property"`
	Value    types.String `tfsdk:"value"`
	ParentID types.Int64  `tfsdk:"parent_id"`
	EntryID  types.Int64  `tfsdk:"entry_id"`
	Const    types.String `tfsdk:"const"`
}

func (r *dialogValueResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dialog_value"
}

func (r *dialogValueResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a single selectable value of a dialog / dialog+ attribute via `cmdb.dialog.*`. Note that many built-in i-doit dialogs are read-only through the API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "`<category>/<property>/<entry_id>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"category": schema.StringAttribute{
				MarkdownDescription: "Category constant that owns the attribute, e.g. `C__CATG__MODEL`. Changing this forces a new value.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"property": schema.StringAttribute{
				MarkdownDescription: "Attribute key within the category, e.g. `manufacturer`. Changing this forces a new value.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "Display text of the dialog value.",
				Required:            true,
			},
			"parent_id": schema.Int64Attribute{
				MarkdownDescription: "Parent value id for a hierarchical dialog+ attribute. Changing this forces a new value.",
				Optional:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"entry_id": schema.Int64Attribute{
				MarkdownDescription: "Numeric id of the dialog value.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"const": schema.StringAttribute{
				MarkdownDescription: "Constant assigned by i-doit, if any.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *dialogValueResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *dialogValueResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dialogValueResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := r.client.CreateDialogValue(ctx,
		plan.Category.ValueString(),
		plan.Property.ValueString(),
		plan.Value.ValueString(),
		plan.ParentID.ValueInt64(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create dialog value", err.Error())
		return
	}
	tflog.Info(ctx, "created dialog value", map[string]any{
		"category": plan.Category.ValueString(), "property": plan.Property.ValueString(), "id": id,
	})

	plan.EntryID = types.Int64Value(id)
	plan.ID = types.StringValue(fmt.Sprintf("%s/%s/%d", plan.Category.ValueString(), plan.Property.ValueString(), id))
	if err := r.refresh(ctx, &plan, id, false); err != nil {
		resp.Diagnostics.AddError("Unable to read back dialog value", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dialogValueResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dialogValueResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.EntryID.ValueInt64()

	val, err := r.findValue(ctx, &state, id)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read dialog value", err.Error())
		return
	}
	if val == nil {
		tflog.Warn(ctx, "dialog value gone, removing from state", map[string]any{"id": id})
		resp.State.RemoveResource(ctx)
		return
	}
	if err := r.refresh(ctx, &state, id, true); err != nil {
		resp.Diagnostics.AddError("Unable to read dialog value", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dialogValueResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state dialogValueResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.EntryID.ValueInt64()
	plan.ID = state.ID
	plan.EntryID = state.EntryID

	if plan.Value.ValueString() != state.Value.ValueString() {
		if err := r.client.UpdateDialog(ctx, plan.Category.ValueString(), plan.Property.ValueString(), id, plan.Value.ValueString()); err != nil {
			resp.Diagnostics.AddError("Unable to update dialog value", err.Error())
			return
		}
	}
	if err := r.refresh(ctx, &plan, id, false); err != nil {
		resp.Diagnostics.AddError("Unable to read back dialog value", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dialogValueResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dialogValueResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteDialog(ctx, state.Category.ValueString(), state.Property.ValueString(), state.EntryID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Unable to delete dialog value", err.Error())
	}
}

func (r *dialogValueResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import id",
			"Expected `<category>/<property>/<entry_id>`, got "+req.ID)
		return
	}
	entryID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import id", "entry_id must be numeric: "+err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("category"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("property"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("entry_id"), types.Int64Value(entryID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *dialogValueResource) findValue(ctx context.Context, m *dialogValueResourceModel, id int64) (*client.DialogValue, error) {
	vals, err := r.client.ReadDialog(ctx, m.Category.ValueString(), m.Property.ValueString())
	if err != nil {
		return nil, err
	}
	for i := range vals {
		if vals[i].ID.Int64() == id {
			return &vals[i], nil
		}
	}
	return nil, nil
}

func (r *dialogValueResource) refresh(ctx context.Context, m *dialogValueResourceModel, id int64, fromRead bool) error {
	val, err := r.findValue(ctx, m, id)
	if err != nil {
		return err
	}
	if val != nil {
		if c := val.ConstValue(); c != "" {
			m.Const = types.StringValue(c)
		}
		if fromRead {
			if val.Title != "" {
				m.Value = types.StringValue(val.Title)
			}
			if p := val.ParentID.Int64(); p > 0 {
				m.ParentID = types.Int64Value(p)
			}
		}
	}
	if m.Const.IsUnknown() {
		m.Const = types.StringNull()
	}
	return nil
}
