//go:build tools

// Package tools pins the code-generation tooling (tfplugindocs) as an explicit
// module dependency. It is never compiled into the provider binary.
package tools

import (
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
