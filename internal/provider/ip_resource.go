package provider

import (
	"context"
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
	_ resource.Resource                = (*ipResource)(nil)
	_ resource.ResourceWithConfigure   = (*ipResource)(nil)
	_ resource.ResourceWithImportState = (*ipResource)(nil)
)

// NewIPResource is the resource factory registered with the provider.
func NewIPResource() resource.Resource { return &ipResource{} }

type ipResource struct {
	client *client.Client
}

type ipResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ObjectID  types.String `tfsdk:"object_id"`
	NetID     types.String `tfsdk:"net_id"`
	IPAddress types.String `tfsdk:"ipv4_address"`
	Hostname  types.String `tfsdk:"hostname"`
	Primary   types.Bool   `tfsdk:"primary"`
	Active    types.Bool   `tfsdk:"active"`
	EntryID   types.Int64  `tfsdk:"entry_id"`
}

func (m ipResourceModel) objectID() (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(m.ObjectID.ValueString()), 10, 64)
}

func (r *ipResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ip"
}

func (r *ipResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an IPv4 address assignment (`" + catIP + "` category entry) on an existing object, optionally placing it into a Layer-3 net.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Synthetic identifier `<object_id>/<entry_id>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"object_id": schema.StringAttribute{
				MarkdownDescription: "Numeric id of the object the address belongs to (accepts an `idoit_object` `id`). Changing this forces a new entry.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"ipv4_address": schema.StringAttribute{
				MarkdownDescription: "IPv4 address, e.g. `10.0.0.11`.",
				Required:            true,
			},
			"net_id": schema.StringAttribute{
				MarkdownDescription: "Object id of the Layer-3 net (`idoit_layer3_net` `id`) this address belongs to.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"hostname": optionalComputedString("Hostname associated with the address."),
			"primary": schema.BoolAttribute{
				MarkdownDescription: "Mark this as the primary address of the object.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"active": schema.BoolAttribute{
				MarkdownDescription: "Whether the address is active.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"entry_id": schema.Int64Attribute{
				MarkdownDescription: "Numeric id of the category entry as assigned by i-doit.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *ipResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *ipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	objID, err := plan.objectID()
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("object_id"), "Invalid object_id", err.Error())
		return
	}

	entryID, err := r.client.SaveCategory(ctx, objID, catIP, r.buildData(&plan), 0)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create IP entry", err.Error())
		return
	}
	tflog.Info(ctx, "created IP entry", map[string]any{"object_id": objID, "entry_id": entryID})

	plan.EntryID = types.Int64Value(entryID)
	plan.ID = types.StringValue(strconv.FormatInt(objID, 10) + "/" + strconv.FormatInt(entryID, 10))
	r.refresh(ctx, &plan, objID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	objID, err := state.objectID()
	if err != nil {
		resp.Diagnostics.AddError("Invalid object_id in state", err.Error())
		return
	}

	entry, err := findCategoryEntry(ctx, r.client, objID, catIP, state.EntryID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read IP entry", err.Error())
		return
	}
	if entry == nil {
		tflog.Warn(ctx, "IP entry gone, removing from state", map[string]any{"object_id": objID})
		resp.State.RemoveResource(ctx)
		return
	}
	applyIPEntry(&state, entry)
	// Read may surface drift on the Required address; Create/Update must not.
	setStrFromEntry(&state.IPAddress, entry, "ipv4_address")
	state.ID = types.StringValue(strconv.FormatInt(objID, 10) + "/" + strconv.FormatInt(entry.EntryID(), 10))
	state.EntryID = types.Int64Value(entry.EntryID())
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	objID, err := state.objectID()
	if err != nil {
		resp.Diagnostics.AddError("Invalid object_id in state", err.Error())
		return
	}
	entryID := state.EntryID.ValueInt64()

	if _, err := r.client.SaveCategory(ctx, objID, catIP, r.buildData(&plan), entryID); err != nil {
		resp.Diagnostics.AddError("Unable to update IP entry", err.Error())
		return
	}
	plan.EntryID = types.Int64Value(entryID)
	plan.ID = types.StringValue(strconv.FormatInt(objID, 10) + "/" + strconv.FormatInt(entryID, 10))
	r.refresh(ctx, &plan, objID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	objID, err := state.objectID()
	if err != nil {
		resp.Diagnostics.AddError("Invalid object_id in state", err.Error())
		return
	}
	if err := r.client.DeleteCategoryEntry(ctx, objID, catIP, state.EntryID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Unable to delete IP entry", err.Error())
	}
}

func (r *ipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import id", "Expected `<object_id>/<entry_id>`, got "+strconv.Quote(req.ID))
		return
	}
	if _, err := strconv.ParseInt(parts[0], 10, 64); err != nil {
		resp.Diagnostics.AddError("Invalid import id", "object_id must be numeric: "+err.Error())
		return
	}
	entryID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import id", "entry_id must be numeric: "+err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("object_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("entry_id"), entryID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *ipResource) buildData(m *ipResourceModel) map[string]any {
	data := map[string]any{}
	putStr(data, "ipv4_address", m.IPAddress)
	putStr(data, "net", m.NetID)
	putStr(data, "hostname", m.Hostname)
	putBool(data, "primary", m.Primary)
	putBool(data, "active", m.Active)
	return data
}

func (r *ipResource) refresh(ctx context.Context, m *ipResourceModel, objID int64) {
	entry, err := findCategoryEntry(ctx, r.client, objID, catIP, m.EntryID.ValueInt64())
	if err == nil && entry != nil {
		applyIPEntry(m, entry)
	}
	nullUnknownStr(&m.NetID, &m.Hostname)
	nullUnknownBool(&m.Primary, &m.Active)
}

// applyIPEntry refreshes the optional / computed fields from an API entry. It
// deliberately leaves the Required ipv4_address untouched so Create and Update
// never diverge from the plan; Read reconciles that field separately.
func applyIPEntry(m *ipResourceModel, entry map[string]any) {
	setRefFromEntry(&m.NetID, entry, "net")
	setStrFromEntry(&m.Hostname, entry, "hostname")
	setBoolFromEntry(&m.Primary, entry, "primary")
	setBoolFromEntry(&m.Active, entry, "active")
}
