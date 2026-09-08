module github.com/withakedo/terraform-provider-i-doit

go 1.23.0

require (
	github.com/hashicorp/terraform-plugin-framework v1.13.0
	github.com/hashicorp/terraform-plugin-go v0.25.0
	github.com/hashicorp/terraform-plugin-log v0.9.0
	github.com/hashicorp/terraform-plugin-testing v1.11.0
)

// Indirect dependencies are resolved by `go mod tidy`.
