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
	_ datasource.DataSource              = (*objectDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*objectDataSource)(nil)
)

// NewObjectDataSource is the data source factory registered with the provider.
func NewObjectDataSource() datasource.DataSource { return &objectDataSource{} }

type objectDataSource struct {
	client *client.Client
}

type objectDataSourceModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Title     types.String `tfsdk:"title"`
	Type      types.String `tfsdk:"type"`
	SysID     types.String `tfsdk:"sysid"`
	TypeTitle types.String `tfsdk:"type_title"`
	Status    types.String `tfsdk:"status"`
}

func (d *objectDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_object"
}

func (d *objectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a single CMDB object either by its numeric `id` or by `title` (optionally narrowed with `type`).",
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
			"type": schema.StringAttribute{
				MarkdownDescription: "Object type constant used to disambiguate a `title` lookup, e.g. `C__OBJTYPE__SERVER`.",
				Optional:            true,
				Computed:            true,
			},
			"sysid": schema.StringAttribute{
				MarkdownDescription: "SYSID of the object.",
				Computed:            true,
			},
			"type_title": schema.StringAttribute{
				MarkdownDescription: "Localised object type label.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "CMDB status label.",
				Computed:            true,
			},
		},
	}
}

func (d *objectDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data",
			fmt.Sprintf("Expected *client.Client, got %T. This is a provider bug.", req.ProviderData))
		return
	}
	d.client = c
}

func (d *objectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config objectDataSourceModel
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

	var (
		obj *client.Object
		err error
	)

	if hasID {
		obj, err = d.client.ReadObject(ctx, config.ID.ValueInt64())
		if err != nil {
			resp.Diagnostics.AddError("Unable to read i-doit object", err.Error())
			return
		}
		if obj == nil {
			resp.Diagnostics.AddError("Object not found",
				fmt.Sprintf("No object with id %d.", config.ID.ValueInt64()))
			return
		}
	} else {
		filter := client.ObjectsFilter{Title: config.Title.ValueString(), Type: config.Type.ValueString()}
		items, err := d.client.ListObjects(ctx, filter, 0)
		if err != nil {
			resp.Diagnostics.AddError("Unable to list i-doit objects", err.Error())
			return
		}
		matches := filterExactTitle(items, config.Title.ValueString())
		switch len(matches) {
		case 0:
			resp.Diagnostics.AddError("Object not found",
				fmt.Sprintf("No object titled %q.", config.Title.ValueString()))
			return
		case 1:
			// fall through
		default:
			resp.Diagnostics.AddError("Ambiguous object lookup",
				fmt.Sprintf("%d objects are titled %q; narrow with `type`.", len(matches), config.Title.ValueString()))
			return
		}
		obj, err = d.client.ReadObject(ctx, matches[0].ID.Int64())
		if err != nil {
			resp.Diagnostics.AddError("Unable to read i-doit object", err.Error())
			return
		}
		if obj == nil {
			resp.Diagnostics.AddError("Object not found", "The matched object disappeared during lookup.")
			return
		}
	}

	config.ID = types.Int64Value(obj.ID.Int64())
	config.Title = types.StringValue(obj.Title)
	if obj.TypeConst != "" {
		config.Type = types.StringValue(obj.TypeConst)
	} else if config.Type.IsNull() {
		config.Type = types.StringNull()
	}
	config.SysID = types.StringValue(obj.SysID)
	config.TypeTitle = types.StringValue(obj.TypeTitle)
	config.Status = types.StringValue(obj.StatusLabel())

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func filterExactTitle(items []client.ObjectListItem, title string) []client.ObjectListItem {
	var out []client.ObjectListItem
	for _, it := range items {
		if it.Title == title {
			out = append(out, it)
		}
	}
	return out
}
