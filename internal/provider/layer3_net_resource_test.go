package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccLayer3NetResource exercises the create / read / update / import / delete
// lifecycle of idoit_layer3_net together with an idoit_ip placed into it.
// Requires a reachable i-doit instance and TF_ACC=1.
func TestAccLayer3NetResource(t *testing.T) {
	title := acctestObjectTitle("l3net")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccLayer3NetConfig(title, "24"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("idoit_layer3_net.test", "address", "10.20.30.0"),
					resource.TestCheckResourceAttr("idoit_layer3_net.test", "cidr_suffix", "24"),
					resource.TestCheckResourceAttr("idoit_layer3_net.test", "type", "ipv4"),
					resource.TestCheckResourceAttrSet("idoit_layer3_net.test", "id"),
					resource.TestCheckResourceAttrPair(
						"idoit_ip.test", "net_id", "idoit_layer3_net.test", "id"),
				),
			},
			{
				ResourceName:      "idoit_layer3_net.test",
				ImportState:       true,
				ImportStateVerify: false,
			},
			{
				Config: testAccLayer3NetConfig(title, "25"),
				Check:  resource.TestCheckResourceAttr("idoit_layer3_net.test", "cidr_suffix", "25"),
			},
		},
	})
}

func testAccLayer3NetConfig(title, suffix string) string {
	return fmt.Sprintf(`
%s

resource "idoit_layer3_net" "test" {
  title            = %q
  address          = "10.20.30.0"
  cidr_suffix      = %q
  dns_server       = "10.20.30.1"
  purge_on_destroy = true
}

resource "idoit_object" "host" {
  type             = "C__OBJTYPE__SERVER"
  title            = "%s-host"
  purge_on_destroy = true
}

resource "idoit_ip" "test" {
  object_id    = idoit_object.host.id
  net_id       = idoit_layer3_net.test.id
  ipv4_address = "10.20.30.10"
  hostname     = "%s-host"
  primary      = true
}
`, providerConfig, title, suffix, title, title)
}
