package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccObjectResource is a smoke test for the full create / read / update /
// import / delete lifecycle of idoit_object. It requires a reachable i-doit
// instance and TF_ACC=1.
func TestAccObjectResource(t *testing.T) {
	title := acctestObjectTitle("object")
	titleUpdated := title + "-renamed"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccObjectResourceConfig("C__OBJTYPE__SERVER", title),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("idoit_object.test", "type", "C__OBJTYPE__SERVER"),
					resource.TestCheckResourceAttr("idoit_object.test", "title", title),
					resource.TestCheckResourceAttrSet("idoit_object.test", "id"),
					resource.TestCheckResourceAttrSet("idoit_object.test", "sysid"),
				),
			},
			{
				// cmdb.object.read does not return the object type constant on
				// every i-doit version, so `type` is not verified on import.
				ResourceName:            "idoit_object.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"purge_on_destroy", "type"},
			},
			{
				Config: testAccObjectResourceConfig("C__OBJTYPE__SERVER", titleUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("idoit_object.test", "title", titleUpdated),
				),
			},
		},
	})
}

func testAccObjectResourceConfig(objType, title string) string {
	return fmt.Sprintf(`
%s

resource "idoit_object" "test" {
  type             = %q
  title            = %q
  purge_on_destroy = true
}
`, providerConfig, objType, title)
}
