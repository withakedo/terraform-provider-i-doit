package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccCategoryEntryResource exercises idoit_category_entry against the global
// "model" category (C__CATG__MODEL) of a throwaway server object. Requires a
// reachable i-doit instance and TF_ACC=1.
func TestAccCategoryEntryResource(t *testing.T) {
	title := acctestObjectTitle("catentry")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCategoryEntryConfig(title, "tf-acc-model-1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"idoit_category_entry.model", "object_id",
						"idoit_object.host", "id"),
					resource.TestCheckResourceAttr("idoit_category_entry.model", "category", "C__CATG__MODEL"),
					resource.TestCheckResourceAttr("idoit_category_entry.model", "data.model", "tf-acc-model-1"),
					resource.TestCheckResourceAttrSet("idoit_category_entry.model", "entry_id"),
				),
			},
			{
				Config: testAccCategoryEntryConfig(title, "tf-acc-model-2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("idoit_category_entry.model", "data.model", "tf-acc-model-2"),
				),
			},
			{
				ResourceName:      "idoit_category_entry.model",
				ImportState:       true,
				ImportStateVerify: false,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["idoit_category_entry.model"]
					if !ok {
						return "", fmt.Errorf("resource idoit_category_entry.model not found in state")
					}
					return rs.Primary.ID, nil
				},
			},
		},
	})
}

func testAccCategoryEntryConfig(title, model string) string {
	return fmt.Sprintf(`
%s

resource "idoit_object" "host" {
  type             = "C__OBJTYPE__SERVER"
  title            = %q
  purge_on_destroy = true
}

resource "idoit_category_entry" "model" {
  object_id = idoit_object.host.id
  category  = "C__CATG__MODEL"

  data = {
    model = %q
  }
}
`, providerConfig, title, model)
}
