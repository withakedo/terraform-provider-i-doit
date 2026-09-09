package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/withakedo/terraform-provider-i-doit/internal/client"
)

var (
	_ datasource.DataSource              = (*layer3NetDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*layer3NetDataSource)(nil)
)

// NewLayer3NetDataSource is the data source factory registered with the provider.
func NewLayer3NetDataSource() datasource.DataSource { return &layer3NetDataSource{} }

type layer3NetDataSource struct {
	client *client.Client
}

type layer3NetDataSourceModel struct {
	ID         types.Int64  `tfsdk:"id"`
	Title      types.String `tfsdk:"title"`
	Type       types.String `tfsdk:"type"`
	Address    types.String `tfsdk:"address"`
	CIDRSuffix types.String `tfsdk:"cidr_suffix"`
	DNSServer  types.String `tfsdk:"dns_server"`
	DNSDomain  types.String `tfsdk:"dns_domain"`
	RangeFrom  types.String `tfsdk:"range_from"`
	RangeTo    types.String `tfsdk:"range_to"`
	SysID      types.String `tfsdk:"sysid"`
}

func (d *layer3NetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_layer3_net"
}

func (d *layer3NetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a Layer-3 net by numeric `id` or by `title` and returns its `" + catL3Net + "` fields.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "Numeric object id. Provide this or `title`.",
				Optional:            true,
				Computed:            true,
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "Object title. Provide this or `id`.",
				Optional:            true,
				Computed:            true,
			},
			"type":        schema.StringAttribute{Computed: true, MarkdownDescription: "`ipv4` or `ipv6`."},
			"address":     schema.StringAttribute{Computed: true, MarkdownDescription: "Network address."},
			"cidr_suffix": schema.StringAttribute{Computed: true, MarkdownDescription: "CIDR suffix."},
			"dns_server":  schema.StringAttribute{Computed: true, MarkdownDescription: "DNS server."},
			"dns_domain":  schema.StringAttribute{Computed: true, MarkdownDescription: "DNS domain."},
			"range_from":  schema.StringAttribute{Computed: true, MarkdownDescription: "Start of the address range."},
			"range_to":    schema.StringAttribute{Computed: true, MarkdownDescription: "End of the address range."},
			"sysid":       schema.StringAttribute{Computed: true, MarkdownDescription: "SYSID."},
		},
	}
}

func (d *layer3NetDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *layer3NetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config layer3NetDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasID := !config.ID.IsNull() && !config.ID.IsUnknown() && config.ID.ValueInt64() > 0
	hasTitle := !config.Title.IsNull() && !config.Title.IsUnknown() && config.Title.ValueString() != ""
	if hasID == hasTitle {
		resp.Diagnostics.AddError("Ambiguous lookup", "Set exactly one of `id` or `title`.")
		return
	}

	var id int64
	if hasID {
		id = config.ID.ValueInt64()
	} else {
		items, err := d.client.ListObjects(ctx, client.ObjectsFilter{
			Type: objTypeLayer3Net, Title: config.Title.ValueString(),
		}, 0)
		if err != nil {
			resp.Diagnostics.AddError("Unable to list Layer-3 nets", err.Error())
			return
		}
		var matches []client.ObjectListItem
		for _, it := range items {
			if it.Title == config.Title.ValueString() {
				matches = append(matches, it)
			}
		}
		switch len(matches) {
		case 0:
			resp.Diagnostics.AddError("Layer-3 net not found",
				fmt.Sprintf("No Layer-3 net titled %q.", config.Title.ValueString()))
			return
		case 1:
			id = matches[0].ID.Int64()
		default:
			resp.Diagnostics.AddError("Ambiguous lookup",
				fmt.Sprintf("%d Layer-3 nets are titled %q.", len(matches), config.Title.ValueString()))
			return
		}
	}

	obj, err := d.client.ReadObject(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Layer-3 net", err.Error())
		return
	}
	if obj == nil {
		resp.Diagnostics.AddError("Layer-3 net not found", fmt.Sprintf("No object with id %d.", id))
		return
	}

	config.ID = types.Int64Value(id)
	config.Title = types.StringValue(obj.Title)
	config.SysID = types.StringValue(obj.SysID)

	e, err := firstCategoryEntry(ctx, d.client, id, catL3Net)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Net category", err.Error())
		return
	}
	if e != nil {
		config.Type = types.StringValue(constToNetType(e))
		config.Address = stringFromEntryOrNull(e, "address")
		config.CIDRSuffix = stringFromEntryOrNull(e, "cidr_suffix")
		config.DNSServer = stringFromEntryOrNull(e, "dns_server")
		config.DNSDomain = stringFromEntryOrNull(e, "dns_domain")
		config.RangeFrom = stringFromEntryOrNull(e, "range_from")
		config.RangeTo = stringFromEntryOrNull(e, "range_to")
	} else {
		config.Type = types.StringNull()
		config.Address = types.StringNull()
		config.CIDRSuffix = types.StringNull()
		config.DNSServer = types.StringNull()
		config.DNSDomain = types.StringNull()
		config.RangeFrom = types.StringNull()
		config.RangeTo = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

// stringFromEntryOrNull returns the coerced field value, or a null string when
// the field is absent / unrepresentable.
func stringFromEntryOrNull(entry map[string]any, key string) types.String {
	if s, ok := entryStr(entry, key); ok {
		return types.StringValue(s)
	}
	return types.StringNull()
}
