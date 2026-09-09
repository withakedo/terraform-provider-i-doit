package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"

	"github.com/withakedo/terraform-provider-i-doit/internal/client"
)

// clientFromProviderData unwraps the *client.Client stored on the provider,
// recording a diagnostic (and returning nil) on the unexpected-type path.
func clientFromProviderData(pd any, diags *diag.Diagnostics) *client.Client {
	if pd == nil {
		return nil
	}
	c, ok := pd.(*client.Client)
	if !ok {
		diags.AddError("Unexpected provider data",
			fmt.Sprintf("Expected *client.Client, got %T. This is a provider bug.", pd))
		return nil
	}
	return c
}

// optionalComputedString is the schema for a free-form string attribute that the
// user may set and that is otherwise refreshed from the API.
func optionalComputedString(desc string) schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: desc,
		Optional:            true,
		Computed:            true,
		PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
}

// deleteObject archives (or, when purge is true, permanently purges) a CMDB
// object and records any failure as a diagnostic.
func deleteObject(ctx context.Context, c *client.Client, id int64, purge bool, diags *diag.Diagnostics) {
	var err error
	verb := "archive"
	if purge {
		verb = "purge"
		err = c.PurgeObject(ctx, id)
	} else {
		err = c.ArchiveObject(ctx, id)
	}
	if err != nil {
		diags.AddError("Unable to "+verb+" i-doit object", err.Error())
	}
}
