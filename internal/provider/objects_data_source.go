package provider

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/withakedo/terraform-provider-i-doit/internal/client"
)

var (
	_ datasource.DataSource              = (*objectsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*objectsDataSource)(nil)
)

// NewObjectsDataSource is the data source factory registered with the provider.
func NewObjectsDataSource() datasource.DataSource { return &objectsDataSource{} }

type objectsDataSource struct {
	client *client.Client
}

type objectsDataSourceModel struct {
	Type    types.String       `tfsdk:"type"`
	Title   types.String       `tfsdk:"title"`
	Limit   types.Int64        `tfsdk:"limit"`
	ID      types.String       `tfsdk:"id"`
	Objects []objectsItemModel `tfsdk:"objects"`
}

type objectsItemModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Title     types.String `tfsdk:"title"`
	SysID     types.String `tfsdk:"sysid"`
	Type      types.String `tfsdk:"type"`
	TypeTitle types.String `tfsdk:"type_title"`
	Status    types.String `tfsdk:"status"`
}

func (d *objectsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_objects"
}

func (d *objectsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists CMDB objects filtered by object type and/or a title pattern (`cmdb.objects.read`).",
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				MarkdownDescription: "Object type constant to filter by, e.g. `C__OBJTYPE__SERVER`.",
				Optional:            true,
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "Title filter. i-doit treats `%` as a wildcard, e.g. `web%`.",
				Optional:            true,
			},
			"limit": schema.Int64Attribute{
				MarkdownDescription: "Maximum number of results. `0` or unset means no limit.",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Synthetic identifier derived from the filter.",
				Computed:            true,
			},
			"objects": schema.ListNestedAttribute{
				MarkdownDescription: "Matched objects.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":         schema.Int64Attribute{Computed: true, MarkdownDescription: "Numeric object id."},
						"title":      schema.StringAttribute{Computed: true, MarkdownDescription: "Object title."},
						"sysid":      schema.StringAttribute{Computed: true, MarkdownDescription: "SYSID."},
						"type":       schema.StringAttribute{Computed: true, MarkdownDescription: "Object type constant."},
						"type_title": schema.StringAttribute{Computed: true, MarkdownDescription: "Localised object type label."},
						"status":     schema.StringAttribute{Computed: true, MarkdownDescription: "Record status."},
					},
				},
			},
		},
	}
}

func (d *objectsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *objectsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config objectsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	filter := client.ObjectsFilter{
		Type:  config.Type.ValueString(),
		Title: config.Title.ValueString(),
	}
	limit := int(config.Limit.ValueInt64())

	items, err := d.client.ListObjects(ctx, filter, limit)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list i-doit objects", err.Error())
		return
	}

	config.Objects = make([]objectsItemModel, 0, len(items))
	for _, it := range items {
		config.Objects = append(config.Objects, objectsItemModel{
			ID:        types.Int64Value(it.ID.Int64()),
			Title:     types.StringValue(it.Title),
			SysID:     types.StringValue(it.SysID),
			Type:      types.StringValue(it.TypeConst),
			TypeTitle: types.StringValue(it.TypeTitle),
			Status:    types.StringValue(fmt.Sprintf("%d", it.Status.Int64())),
		})
	}

	config.ID = types.StringValue(fmt.Sprintf("%x",
		sha256.Sum256([]byte(filter.Type+"|"+filter.Title+"|"+fmt.Sprint(limit)))))

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
