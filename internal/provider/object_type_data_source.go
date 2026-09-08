package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/withakedo/terraform-provider-i-doit/internal/client"
)

var (
	_ datasource.DataSource              = (*objectTypeDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*objectTypeDataSource)(nil)
)

// NewObjectTypeDataSource is the data source factory registered with the provider.
func NewObjectTypeDataSource() datasource.DataSource { return &objectTypeDataSource{} }

type objectTypeDataSource struct {
	client *client.Client
}

type objectTypeDataSourceModel struct {
	Const       types.String `tfsdk:"const"`
	ID          types.Int64  `tfsdk:"id"`
	Title       types.String `tfsdk:"title"`
	TitleLang   types.String `tfsdk:"title_lang"`
	ObjectCount types.Int64  `tfsdk:"object_count"`
}

func (d *objectTypeDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_object_type"
}

func (d *objectTypeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Resolves an object type between its constant, numeric id and title (`cmdb.object_types`). Provide exactly one of `const`, `id` or `title`.",
		Attributes: map[string]schema.Attribute{
			"const": schema.StringAttribute{
				MarkdownDescription: "Object type constant, e.g. `C__OBJTYPE__SERVER`.",
				Optional:            true,
				Computed:            true,
			},
			"id": schema.Int64Attribute{
				MarkdownDescription: "Numeric object type id.",
				Optional:            true,
				Computed:            true,
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "Object type title.",
				Optional:            true,
				Computed:            true,
			},
			"title_lang": schema.StringAttribute{
				MarkdownDescription: "Localised object type title.",
				Computed:            true,
			},
			"object_count": schema.Int64Attribute{
				MarkdownDescription: "Number of objects of this type.",
				Computed:            true,
			},
		},
	}
}

func (d *objectTypeDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *objectTypeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config objectTypeDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wantConst := strings.TrimSpace(config.Const.ValueString())
	wantTitle := strings.TrimSpace(config.Title.ValueString())
	wantID := config.ID.ValueInt64()
	hasID := !config.ID.IsNull() && !config.ID.IsUnknown() && wantID > 0

	set := 0
	if wantConst != "" {
		set++
	}
	if wantTitle != "" {
		set++
	}
	if hasID {
		set++
	}
	if set != 1 {
		resp.Diagnostics.AddError("Ambiguous lookup", "Set exactly one of `const`, `id` or `title`.")
		return
	}

	types_, err := d.client.ListObjectTypes(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list i-doit object types", err.Error())
		return
	}

	var matches []client.ObjectType
	for _, t := range types_ {
		switch {
		case wantConst != "" && strings.EqualFold(t.Const, wantConst):
			matches = append(matches, t)
		case hasID && t.ID.Int64() == wantID:
			matches = append(matches, t)
		case wantTitle != "" && strings.EqualFold(t.Title, wantTitle):
			matches = append(matches, t)
		}
	}

	switch len(matches) {
	case 0:
		resp.Diagnostics.AddError("Object type not found", "No object type matched the given filter.")
		return
	case 1:
	default:
		resp.Diagnostics.AddError("Ambiguous object type lookup",
			fmt.Sprintf("%d object types matched the given filter.", len(matches)))
		return
	}

	m := matches[0]
	config.Const = types.StringValue(m.Const)
	config.ID = types.Int64Value(m.ID.Int64())
	config.Title = types.StringValue(m.Title)
	config.TitleLang = types.StringValue(m.TitleLang)
	config.ObjectCount = types.Int64Value(m.Count())

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
