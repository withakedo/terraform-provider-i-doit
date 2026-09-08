package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories wires the in-process provider into the
// terraform-plugin-testing harness.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"idoit": providerserver.NewProtocol6WithError(New("test")()),
}

// providerConfig relies on IDOIT_URL / IDOIT_APIKEY (and optionally
// IDOIT_USERNAME / IDOIT_PASSWORD) being present in the environment.
//
// Example against the public demo instance:
//
//	export IDOIT_URL="https://demo.i-doit.com"
//	export IDOIT_APIKEY="<demo api key>"
//	export TF_ACC=1
const providerConfig = `
provider "idoit" {
  request_timeout = 30
  max_retries     = 2
}
`

func testAccPreCheck(t *testing.T) {
	t.Helper()
	for _, key := range []string{"IDOIT_URL", "IDOIT_APIKEY"} {
		if os.Getenv(key) == "" {
			t.Fatalf("%s must be set for acceptance tests", key)
		}
	}
}

// acctestObjectTitle returns a unique title so parallel / repeated runs do not
// collide inside the shared demo CMDB.
func acctestObjectTitle(prefix string) string {
	return fmt.Sprintf("tf-acc-%s-%d", prefix, os.Getpid())
}
