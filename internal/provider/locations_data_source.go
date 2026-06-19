package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/palmahost/terraform-provider-palmahost/internal/client"
)

var (
	_ datasource.DataSource              = (*locationsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*locationsDataSource)(nil)
)

// NewLocationsDataSource registers the palmahost_locations data source.
func NewLocationsDataSource() datasource.DataSource { return &locationsDataSource{} }

type locationsDataSource struct{ c *client.Client }

type apiLocation struct {
	ID      string `json:"id"`
	Code    string `json:"code"`
	Name    string `json:"name"`
	Country string `json:"country"`
	Active  bool   `json:"active"`
}

type locationModel struct {
	ID      types.String `tfsdk:"id"`
	Code    types.String `tfsdk:"code"`
	Name    types.String `tfsdk:"name"`
	Country types.String `tfsdk:"country"`
	Active  types.Bool   `tfsdk:"active"`
}

type locationsModel struct {
	Locations []locationModel `tfsdk:"locations"`
}

func (d *locationsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_locations"
}

func (d *locationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Deployment locations (regions). Use a location `code` (e.g. `mad`) or `country` (e.g. `ES`) with `palmahost_compute_vm`.",
		Attributes: map[string]schema.Attribute{
			"locations": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":      schema.StringAttribute{Computed: true},
						"code":    schema.StringAttribute{Computed: true},
						"name":    schema.StringAttribute{Computed: true},
						"country": schema.StringAttribute{Computed: true},
						"active":  schema.BoolAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *locationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.c, _ = req.ProviderData.(*client.Client)
}

func (d *locationsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	var locs []apiLocation
	if err := d.c.Do(ctx, "GET", "/locations", nil, &locs); err != nil {
		resp.Diagnostics.AddError("Could not read locations", err.Error())
		return
	}
	out := locationsModel{Locations: make([]locationModel, 0, len(locs))}
	for _, l := range locs {
		out.Locations = append(out.Locations, locationModel{
			ID:      types.StringValue(l.ID),
			Code:    types.StringValue(l.Code),
			Name:    types.StringValue(l.Name),
			Country: types.StringValue(l.Country),
			Active:  types.BoolValue(l.Active),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}
