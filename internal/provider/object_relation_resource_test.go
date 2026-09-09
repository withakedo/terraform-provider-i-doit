package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccObjectRelationResource exercises the create / read / update / import /
// delete lifecycle of idoit_object_relation between two throwaway objects.
// Requires a reachable i-doit instance and TF_ACC=1.
func TestAccObjectRelationResource(t *testing.T) {
	title := acctestObjectTitle("rel")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccObjectRelationConfig(title, "first"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"idoit_object_relation.test", "master_object_id",
						"idoit_object.master", "id"),
					resource.TestCheckResourceAttrPair(
						"idoit_object_relation.test", "slave_object_id",
						"idoit_object.slave", "id"),
					resource.TestCheckResourceAttrSet("idoit_object_relation.test", "entry_id"),
				),
			},
			{
				Config: testAccObjectRelationConfig(title, "second"),
				Check:  resource.TestCheckResourceAttr("idoit_object_relation.test", "description", "second"),
			},
			{
				ResourceName:      "idoit_object_relation.test",
				ImportState:       true,
				ImportStateVerify: false,
			},
		},
	})
}

func testAccObjectRelationConfig(title, desc string) string {
	return fmt.Sprintf(`
%s

resource "idoit_object" "master" {
  type             = "C__OBJTYPE__APPLICATION"
  title            = "%s-master"
  purge_on_destroy = true
}

resource "idoit_object" "slave" {
  type             = "C__OBJTYPE__SERVER"
  title            = "%s-slave"
  purge_on_destroy = true
}

resource "idoit_object_relation" "test" {
  master_object_id = idoit_object.master.id
  slave_object_id  = idoit_object.slave.id
  relation_type    = "C__RELATION_TYPE__SOFTWARE"
  description      = %q
}
`, providerConfig, title, title, desc)
}
