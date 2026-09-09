// Package provider implements the Terraform provider for i-doit, built on the
// terraform-plugin-framework.
package provider

import (
	"context"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/withakedo/terraform-provider-i-doit/internal/client"
)

// Ensure the provider satisfies the framework interfaces.
var _ provider.Provider = (*idoitProvider)(nil)

const (
	envURL           = "IDOIT_URL"
	envAPIKey        = "IDOIT_APIKEY"
	envUsername      = "IDOIT_USERNAME"
	envPassword      = "IDOIT_PASSWORD"
	envLanguage      = "IDOIT_LANGUAGE"
	envCACert        = "IDOIT_CA_CERT"
	envClientCert    = "IDOIT_CLIENT_CERT"
	envClientKey     = "IDOIT_CLIENT_KEY"
	envTLSServerName = "IDOIT_TLS_SERVER_NAME"

	defaultRequestTimeout = 60
	defaultMaxRetries     = 3
	defaultMaxConcurrent  = 10
	defaultLanguage       = "en"
)

type idoitProvider struct {
	version string
}

// New returns the provider factory used by main.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &idoitProvider{version: version}
	}
}

type idoitProviderModel struct {
	URL                   types.String `tfsdk:"url"`
	APIKey                types.String `tfsdk:"apikey"`
	Username              types.String `tfsdk:"username"`
	Password              types.String `tfsdk:"password"`
	RequestTimeout        types.Int64  `tfsdk:"request_timeout"`
	MaxRetries            types.Int64  `tfsdk:"max_retries"`
	MaxConcurrentRequests types.Int64  `tfsdk:"max_concurrent_requests"`
	InsecureSkipVerify    types.Bool   `tfsdk:"insecure_skip_verify"`
	CACert                types.String `tfsdk:"ca_cert"`
	ClientCert            types.String `tfsdk:"client_cert"`
	ClientKey             types.String `tfsdk:"client_key"`
	TLSServerName         types.String `tfsdk:"tls_server_name"`
	Language              types.String `tfsdk:"language"`
}

func (p *idoitProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "idoit"
	resp.Version = p.version
}

func (p *idoitProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The i-doit provider manages CMDB objects and their category fields through the i-doit JSON-RPC API.",
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				MarkdownDescription: "Base URL of the i-doit installation, without the `/src/jsonrpc.php` suffix. Can also be set with the `" + envURL + "` environment variable.",
				Optional:            true,
			},
			"apikey": schema.StringAttribute{
				MarkdownDescription: "API key configured under *Administration > Interfaces / external data > JSON-RPC API* in i-doit. Can also be set with the `" + envAPIKey + "` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Username for session authentication (`idoit.login`). Optional; the API key alone is sufficient for most setups. Can also be set with the `" + envUsername + "` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password for session authentication (`idoit.login`). Required when `username` is set. Can also be set with the `" + envPassword + "` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"request_timeout": schema.Int64Attribute{
				MarkdownDescription: "Per-request timeout in seconds. Defaults to `60`.",
				Optional:            true,
			},
			"max_retries": schema.Int64Attribute{
				MarkdownDescription: "Number of retries with exponential backoff for transient failures (network errors, HTTP 429/5xx). Defaults to `3`.",
				Optional:            true,
			},
			"max_concurrent_requests": schema.Int64Attribute{
				MarkdownDescription: "Maximum number of in-flight HTTP requests against the i-doit API. `0` means unlimited. Defaults to `10`.",
				Optional:            true,
			},
			"insecure_skip_verify": schema.BoolAttribute{
				MarkdownDescription: "Disable TLS certificate verification. Defaults to `false`. Use only against trusted test instances.",
				Optional:            true,
			},
			"ca_cert": schema.StringAttribute{
				MarkdownDescription: "Custom CA certificate(s) to trust for the i-doit endpoint, as inline PEM data or a path to a PEM file. Appended to the system trust store. Can also be set with the `" + envCACert + "` environment variable.",
				Optional:            true,
			},
			"client_cert": schema.StringAttribute{
				MarkdownDescription: "Client certificate for mutual TLS, as inline PEM data or a path to a PEM file. Requires `client_key`. Can also be set with the `" + envClientCert + "` environment variable.",
				Optional:            true,
			},
			"client_key": schema.StringAttribute{
				MarkdownDescription: "Private key for `client_cert`, as inline PEM data or a path to a PEM file. Can also be set with the `" + envClientKey + "` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"tls_server_name": schema.StringAttribute{
				MarkdownDescription: "Overrides the server name used for SNI and certificate verification. Can also be set with the `" + envTLSServerName + "` environment variable.",
				Optional:            true,
			},
			"language": schema.StringAttribute{
				MarkdownDescription: "Language passed to the API (`en`, `de`, ...). Defaults to `en`. Can also be set with the `" + envLanguage + "` environment variable.",
				Optional:            true,
			},
		},
	}
}

func (p *idoitProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config idoitProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.URL.IsUnknown() {
		resp.Diagnostics.AddAttributeError(path.Root("url"), "Unknown i-doit URL",
			"The provider cannot be configured with an unknown value for url.")
	}
	if config.APIKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(path.Root("apikey"), "Unknown i-doit API key",
			"The provider cannot be configured with an unknown value for apikey.")
	}
	if resp.Diagnostics.HasError() {
		return
	}

	url := stringValue(config.URL, envURL, "")
	apiKey := stringValue(config.APIKey, envAPIKey, "")
	username := stringValue(config.Username, envUsername, "")
	password := stringValue(config.Password, envPassword, "")
	language := stringValue(config.Language, envLanguage, defaultLanguage)
	caCert := stringValue(config.CACert, envCACert, "")
	clientCert := stringValue(config.ClientCert, envClientCert, "")
	clientKey := stringValue(config.ClientKey, envClientKey, "")
	tlsServerName := stringValue(config.TLSServerName, envTLSServerName, "")

	timeout := int64Value(config.RequestTimeout, defaultRequestTimeout)
	retries := int64Value(config.MaxRetries, defaultMaxRetries)
	maxConcurrent := int64Value(config.MaxConcurrentRequests, defaultMaxConcurrent)
	insecure := config.InsecureSkipVerify.ValueBool()

	if url == "" {
		resp.Diagnostics.AddAttributeError(path.Root("url"), "Missing i-doit URL",
			"Set the url argument or the "+envURL+" environment variable.")
	}
	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(path.Root("apikey"), "Missing i-doit API key",
			"Set the apikey argument or the "+envAPIKey+" environment variable.")
	}
	if (username == "") != (password == "") {
		resp.Diagnostics.AddError("Incomplete session credentials",
			"username and password must be set together for session authentication.")
	}
	if timeout <= 0 {
		resp.Diagnostics.AddAttributeError(path.Root("request_timeout"), "Invalid request_timeout",
			"request_timeout must be a positive number of seconds.")
	}
	if retries < 0 {
		resp.Diagnostics.AddAttributeError(path.Root("max_retries"), "Invalid max_retries",
			"max_retries must be zero or greater.")
	}
	if maxConcurrent < 0 {
		resp.Diagnostics.AddAttributeError(path.Root("max_concurrent_requests"), "Invalid max_concurrent_requests",
			"max_concurrent_requests must be zero or greater.")
	}
	if (clientCert == "") != (clientKey == "") {
		resp.Diagnostics.AddError("Incomplete client certificate",
			"client_cert and client_key must be set together for mutual TLS.")
	}
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "idoit_url", url)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "apikey", "password", "client_key")
	tflog.Debug(ctx, "configuring i-doit client")

	c, err := client.New(client.Config{
		URL:                   url,
		APIKey:                apiKey,
		Username:              username,
		Password:              password,
		Language:              language,
		RequestTimeout:        time.Duration(timeout) * time.Second,
		MaxRetries:            int(retries),
		MaxConcurrentRequests: int(maxConcurrent),
		InsecureSkipVerify:    insecure,
		CACert:                caCert,
		ClientCert:            clientCert,
		ClientKey:             clientKey,
		TLSServerName:         tlsServerName,
		UserAgent:             "terraform-provider-i-doit/" + p.version,
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create i-doit client", err.Error())
		return
	}

	if c.HasCredentials() {
		if err := c.Login(ctx); err != nil {
			resp.Diagnostics.AddError("i-doit session login failed",
				"idoit.login was rejected: "+err.Error())
			return
		}
	}

	version, err := c.ReadVersion(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to reach the i-doit API",
			"idoit.version failed against "+c.Endpoint()+": "+err.Error())
		return
	}
	tflog.Info(ctx, "connected to i-doit", map[string]any{"version": version.Version, "type": version.Type})

	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *idoitProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewObjectResource,
		NewCategoryEntryResource,
		NewLayer3NetResource,
		NewLayer2NetResource,
		NewIPResource,
	}
}

func (p *idoitProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewObjectDataSource,
		NewObjectsDataSource,
		NewObjectTypeDataSource,
		NewLayer3NetDataSource,
	}
}

// stringValue resolves a config attribute against an environment variable and a
// static default, in that order of precedence.
func stringValue(attr types.String, envKey, def string) string {
	if !attr.IsNull() && !attr.IsUnknown() && attr.ValueString() != "" {
		return attr.ValueString()
	}
	if envKey != "" {
		if v := os.Getenv(envKey); v != "" {
			return v
		}
	}
	return def
}

func int64Value(attr types.Int64, def int64) int64 {
	if !attr.IsNull() && !attr.IsUnknown() {
		return attr.ValueInt64()
	}
	return def
}
