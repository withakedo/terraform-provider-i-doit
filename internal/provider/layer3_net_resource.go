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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/withakedo/terraform-provider-i-doit/internal/client"
)

var (
	_ resource.Resource                = (*layer3NetResource)(nil)
	_ resource.ResourceWithConfigure   = (*layer3NetResource)(nil)
	_ resource.ResourceWithImportState = (*layer3NetResource)(nil)
)

// NewLayer3NetResource is the resource factory registered with the provider.
func NewLayer3NetResource() resource.Resource { return &layer3NetResource{} }

type layer3NetResource struct {
	client *client.Client
}

type layer3NetResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Title          types.String `tfsdk:"title"`
	Type           types.String `tfsdk:"type"`
	Address        types.String `tfsdk:"address"`
	CIDRSuffix     types.String `tfsdk:"cidr_suffix"`
	DNSServer      types.String `tfsdk:"dns_server"`
	DNSDomain      types.String `tfsdk:"dns_domain"`
	DefaultGateway types.String `tfsdk:"default_gateway"`
	RangeFrom      types.String `tfsdk:"range_from"`
	RangeTo        types.String `tfsdk:"range_to"`
	Description    types.String `tfsdk:"description"`
	Extra          types.Map    `tfsdk:"extra"`
	PurgeOnDestroy types.Bool   `tfsdk:"purge_on_destroy"`
	SysID          types.String `tfsdk:"sysid"`
}

func (r *layer3NetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_layer3_net"
}

func (r *layer3NetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Layer-3 network in i-doit: a `" + objTypeLayer3Net + "` object together with its `" + catL3Net + "` (Net) category.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Numeric i-doit object id of the Layer-3 net.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "Object title, e.g. `10.0.0.0/24 - servers`.",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "`ipv4` (default) or `ipv6`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("ipv4"),
			},
			"address": schema.StringAttribute{
				MarkdownDescription: "Network address, e.g. `10.0.0.0`.",
				Required:            true,
			},
			"cidr_suffix": schema.StringAttribute{
				MarkdownDescription: "CIDR suffix, e.g. `24`.",
				Required:            true,
			},
			"dns_server":      optionalComputedString("DNS server for the net."),
			"dns_domain":      optionalComputedString("DNS domain for the net."),
			"default_gateway": optionalComputedString("Default gateway address of the net."),
			"range_from":      optionalComputedString("Start of the usable / DHCP address range."),
			"range_to":        optionalComputedString("End of the usable / DHCP address range."),
			"description":     optionalComputedString("Free-text description stored on the net category."),
			"extra": schema.MapAttribute{
				MarkdownDescription: "Additional `" + catL3Net + "` fields passed verbatim to `cmdb.category.save`. Not drift-tracked; use for version-specific keys.",
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

func (r *layer3NetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *layer3NetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan layer3NetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := validateNetType(plan.Type); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("type"), "Invalid type", err.Error())
		return
	}

	id, err := r.client.CreateObject(ctx, objTypeLayer3Net, plan.Title.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Layer-3 net", err.Error())
		return
	}
	plan.ID = types.StringValue(strconv.FormatInt(id, 10))
	tflog.Info(ctx, "created Layer-3 net", map[string]any{"id": id})

	data, diags := r.buildData(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.SaveCategory(ctx, id, catL3Net, data, 0); err != nil {
		resp.Diagnostics.AddError("Unable to write Net category", err.Error())
		return
	}

	if err := r.refresh(ctx, &plan, id, false); err != nil {
		resp.Diagnostics.AddError("Unable to read back Layer-3 net", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *layer3NetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state layer3NetResourceModel
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
		resp.Diagnostics.AddError("Unable to read Layer-3 net", err.Error())
		return
	}
	if obj == nil {
		tflog.Warn(ctx, "Layer-3 net gone, removing from state", map[string]any{"id": id})
		resp.State.RemoveResource(ctx)
		return
	}
	if err := r.refresh(ctx, &state, id, true); err != nil {
		resp.Diagnostics.AddError("Unable to read Net category", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *layer3NetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state layer3NetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := validateNetType(plan.Type); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("type"), "Invalid type", err.Error())
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
	if e, _ := firstCategoryEntry(ctx, r.client, id, catL3Net); e != nil {
		entryID = e.EntryID()
	}
	if _, err := r.client.SaveCategory(ctx, id, catL3Net, data, entryID); err != nil {
		resp.Diagnostics.AddError("Unable to update Net category", err.Error())
		return
	}

	if err := r.refresh(ctx, &plan, id, false); err != nil {
		resp.Diagnostics.AddError("Unable to read back Layer-3 net", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *layer3NetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state layer3NetResourceModel
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

func (r *layer3NetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// buildData assembles the C__CATS__NET data map from the typed fields plus extra.
func (r *layer3NetResource) buildData(ctx context.Context, m *layer3NetResourceModel) (map[string]any, diag.Diagnostics) {
	data := map[string]any{
		"type": netTypeToConst(m.Type.ValueString()),
	}
	putStr(data, "address", m.Address)
	putStr(data, "cidr_suffix", m.CIDRSuffix)
	putStr(data, "dns_server", m.DNSServer)
	putStr(data, "dns_domain", m.DNSDomain)
	putStr(data, "default_gateway", m.DefaultGateway)
	putStr(data, "range_from", m.RangeFrom)
	putStr(data, "range_to", m.RangeTo)
	putStr(data, "description", m.Description)
	return data, mergeExtra(ctx, data, m.Extra)
}

// refresh reloads object + category and updates m in place. When fromRead is
// false (Create / Update) the Required attributes address and cidr_suffix are
// left as planned so the applied state never diverges from the plan; Read passes
// fromRead=true to surface real drift.
func (r *layer3NetResource) refresh(ctx context.Context, m *layer3NetResourceModel, id int64, fromRead bool) error {
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

	e, err := firstCategoryEntry(ctx, r.client, id, catL3Net)
	if err != nil {
		return err
	}
	if e != nil {
		m.Type = types.StringValue(constToNetType(e))
		setStrFromEntry(&m.DNSServer, e, "dns_server")
		setStrFromEntry(&m.DNSDomain, e, "dns_domain")
		setStrFromEntry(&m.DefaultGateway, e, "default_gateway")
		setStrFromEntry(&m.RangeFrom, e, "range_from")
		setStrFromEntry(&m.RangeTo, e, "range_to")
		setStrFromEntry(&m.Description, e, "description")
		if fromRead {
			setStrFromEntry(&m.Address, e, "address")
			setStrFromEntry(&m.CIDRSuffix, e, "cidr_suffix")
		}
	}
	nullUnknownStr(&m.SysID, &m.Type, &m.DNSServer, &m.DNSDomain, &m.DefaultGateway,
		&m.RangeFrom, &m.RangeTo, &m.Description)
	return nil
}

func validateNetType(v types.String) error {
	if v.IsNull() || v.IsUnknown() || v.ValueString() == "" {
		return nil
	}
	switch strings.ToLower(v.ValueString()) {
	case "ipv4", "ipv6":
		return nil
	default:
		return fmt.Errorf("type must be \"ipv4\" or \"ipv6\", got %q", v.ValueString())
	}
}
