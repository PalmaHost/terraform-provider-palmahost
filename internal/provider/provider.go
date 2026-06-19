package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/palmahost/terraform-provider-palmahost/internal/client"
)

// Production API base. The panel and API share the host (Caddy routes /v1/*),
// so there is no separate api.* subdomain. Dev uses my-dev.palmahost.sh.
const defaultBaseURL = "https://my.palmahost.sh/v1"

var _ provider.Provider = (*palmaProvider)(nil)

type palmaProvider struct{ version string }

// New returns the provider factory used by main.
func New(version string) func() provider.Provider {
	return func() provider.Provider { return &palmaProvider{version: version} }
}

type providerModel struct {
	Token   types.String `tfsdk:"token"`
	BaseURL types.String `tfsdk:"base_url"`
}

func (p *palmaProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "palmahost"
	resp.Version = p.version
}

func (p *palmaProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage [PalmaHost Cloud](https://palmahost.sh) resources: servers, SSH keys, networking and more.",
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "API token (`ph_live_…`). Create one in the panel under Account → API tokens. Falls back to the `PALMAHOST_TOKEN` environment variable.",
			},
			"base_url": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "API base URL including the version prefix. Defaults to `" + defaultBaseURL + "` (or the `PALMAHOST_BASE_URL` environment variable).",
			},
		},
	}
}

func (p *palmaProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	token := cfg.Token.ValueString()
	if token == "" {
		token = os.Getenv("PALMAHOST_TOKEN")
	}
	if token == "" {
		resp.Diagnostics.AddError(
			"Missing API token",
			"Set the provider `token` argument or the PALMAHOST_TOKEN environment variable (create a token under Account → API tokens).",
		)
		return
	}

	baseURL := cfg.BaseURL.ValueString()
	if baseURL == "" {
		baseURL = os.Getenv("PALMAHOST_BASE_URL")
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	c := client.New(baseURL, token)
	resp.ResourceData = c
	resp.DataSourceData = c
}

func (p *palmaProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewComputeVMResource,
		NewServiceResource,
		NewSSHKeyResource,
	}
}

func (p *palmaProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewPlansDataSource,
		NewLocationsDataSource,
	}
}
