package provider

import (
	"context"
	"fmt"
	"sort"
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
	_ resource.Resource                = (*categoryEntryResource)(nil)
	_ resource.ResourceWithConfigure   = (*categoryEntryResource)(nil)
	_ resource.ResourceWithImportState = (*categoryEntryResource)(nil)
)

// NewCategoryEntryResource is the resource factory registered with the provider.
func NewCategoryEntryResource() resource.Resource { return &categoryEntryResource{} }

type categoryEntryResource struct {
	client *client.Client
}

type categoryEntryResourceModel struct {
	ID       types.String `tfsdk:"id"`
	ObjectID types.String `tfsdk:"object_id"`
	Category types.String `tfsdk:"category"`
	Data     types.Map    `tfsdk:"data"`
	EntryID  types.Int64  `tfsdk:"entry_id"`
}

func (m categoryEntryResourceModel) objectID() (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(m.ObjectID.ValueString()), 10, 64)
}

func (r *categoryEntryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_category_entry"
}

func (r *categoryEntryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages one entry of an i-doit category (global `C__CATG__*` or specific `C__CATS__*`) attached to an object. " +
			"Every instance maps to exactly one category entry, so multi-value categories are modelled with multiple resources.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Synthetic identifier `<object_id>/<category>/<entry_id>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"object_id": schema.StringAttribute{
				MarkdownDescription: "Numeric id of the object the entry belongs to (accepts the `id` of an `idoit_object` resource). Changing this forces a new entry.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"category": schema.StringAttribute{
				MarkdownDescription: "Category constant, e.g. `C__CATG__IP` or `C__CATG__MODEL`. Changing this forces a new entry.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"data": schema.MapAttribute{
				MarkdownDescription: "Category field values keyed by their attribute name. Values are sent as-is to `cmdb.category.save` and read back with `cmdb.category.read`. " +
					"i-doit may normalise stored values (numbers, dialog titles); the value that i-doit reports is written to state, so a small post-apply diff can occur for such fields. For dialog attributes provide the value title.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"entry_id": schema.Int64Attribute{
				MarkdownDescription: "Numeric id of the category entry as assigned by i-doit.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *categoryEntryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *categoryEntryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan categoryEntryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	objID, err := plan.objectID()
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("object_id"), "Invalid object_id", err.Error())
		return
	}

	if plan.Data.IsNull() || plan.Data.IsUnknown() {
		resp.Diagnostics.AddAttributeError(path.Root("data"), "Missing data",
			"`data` must contain at least one category field.")
		return
	}

	data, diags := mapToStringMap(ctx, plan.Data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	category := plan.Category.ValueString()

	entryID, err := r.client.SaveCategory(ctx, objID, category, toAnyMap(data), 0)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create i-doit category entry", err.Error())
		return
	}
	tflog.Info(ctx, "created i-doit category entry", map[string]any{
		"object_id": objID, "category": category, "entry_id": entryID,
	})

	plan.EntryID = types.Int64Value(entryID)
	plan.ID = types.StringValue(categoryEntryID(objID, category, entryID))

	resp.Diagnostics.Append(r.refreshData(ctx, &plan, objID, keysOf(data))...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *categoryEntryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state categoryEntryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	objID, err := state.objectID()
	if err != nil {
		resp.Diagnostics.AddError("Invalid object_id in state", err.Error())
		return
	}
	category := state.Category.ValueString()
	entryID := state.EntryID.ValueInt64()

	entry, err := r.findEntry(ctx, objID, category, entryID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read i-doit category entry", err.Error())
		return
	}
	if entry == nil {
		tflog.Warn(ctx, "i-doit category entry no longer exists, removing from state", map[string]any{
			"object_id": objID, "category": category, "entry_id": entryID,
		})
		resp.State.RemoveResource(ctx)
		return
	}

	prior, _ := mapToStringMap(ctx, state.Data)

	var managed []string
	if state.Data.IsNull() || state.Data.IsUnknown() {
		// Import: adopt every representable field.
		for k := range entry {
			if k == "id" {
				continue
			}
			managed = append(managed, k)
		}
		sort.Strings(managed)
	} else {
		managed = keysOf(prior)
	}

	newData := reconcileData(entry, managed, prior)

	mv, diags := types.MapValueFrom(ctx, types.StringType, newData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Data = mv
	state.EntryID = types.Int64Value(entry.EntryID())
	state.ID = types.StringValue(categoryEntryID(objID, category, entry.EntryID()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *categoryEntryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state categoryEntryResourceModel
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

	data, diags := mapToStringMap(ctx, plan.Data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	category := state.Category.ValueString()
	entryID := state.EntryID.ValueInt64()

	if _, err := r.client.SaveCategory(ctx, objID, category, toAnyMap(data), entryID); err != nil {
		resp.Diagnostics.AddError("Unable to update i-doit category entry", err.Error())
		return
	}

	plan.EntryID = types.Int64Value(entryID)
	plan.ID = types.StringValue(categoryEntryID(objID, category, entryID))

	resp.Diagnostics.Append(r.refreshData(ctx, &plan, objID, keysOf(data))...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *categoryEntryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state categoryEntryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	objID, err := state.objectID()
	if err != nil {
		resp.Diagnostics.AddError("Invalid object_id in state", err.Error())
		return
	}

	if err := r.client.DeleteCategoryEntry(ctx, objID, state.Category.ValueString(), state.EntryID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Unable to delete i-doit category entry", err.Error())
	}
}

func (r *categoryEntryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError("Invalid import id",
			"Expected `<object_id>/<category>/<entry_id>`, got "+strconv.Quote(req.ID))
		return
	}
	if _, err := strconv.ParseInt(parts[0], 10, 64); err != nil {
		resp.Diagnostics.AddError("Invalid import id", "object_id must be numeric: "+err.Error())
		return
	}
	entryID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import id", "entry_id must be numeric: "+err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("object_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("category"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("entry_id"), entryID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// refreshData reads the entry back and rewrites m.Data with the API values for
// the managed keys, so state reflects what i-doit actually stored.
func (r *categoryEntryResource) refreshData(ctx context.Context, m *categoryEntryResourceModel, objID int64, managed []string) diag.Diagnostics {
	entry, err := r.findEntry(ctx, objID, m.Category.ValueString(), m.EntryID.ValueInt64())
	if err != nil || entry == nil {
		// Keep the planned values; Read reconciles on the next refresh.
		return nil
	}

	prior, _ := mapToStringMap(ctx, m.Data)
	newData := reconcileData(entry, managed, prior)

	mv, d := types.MapValueFrom(ctx, types.StringType, newData)
	if d.HasError() {
		return d
	}
	m.Data = mv
	return nil
}

func (r *categoryEntryResource) findEntry(ctx context.Context, objID int64, category string, entryID int64) (client.CategoryEntry, error) {
	entries, err := r.client.ReadCategory(ctx, objID, category)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.EntryID() == entryID {
			return e, nil
		}
	}
	// Single-value categories may not expose a per-entry id; fall back to the
	// sole entry when we did not match by id.
	if len(entries) == 1 {
		return entries[0], nil
	}
	return nil, nil
}
