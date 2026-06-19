package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/palmahost/terraform-provider-palmahost/internal/client"
)

var (
	_ datasource.DataSource              = (*plansDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*plansDataSource)(nil)
)

// NewPlansDataSource registers the palmahost_plans data source.
func NewPlansDataSource() datasource.DataSource { return &plansDataSource{} }

type plansDataSource struct{ c *client.Client }

type planModel struct {
	ID                  types.String `tfsdk:"id"`
	ProductID           types.String `tfsdk:"product_id"`
	Code                types.String `tfsdk:"code"`
	BasePriceCentsMonth types.Int64  `tfsdk:"base_price_cents_month"`
	Popular             types.Bool   `tfsdk:"popular"`
	Active              types.Bool   `tfsdk:"active"`
}

type plansModel struct {
	Plans []planModel `tfsdk:"plans"`
}

type apiPlan struct {
	ID                  string `json:"id"`
	ProductID           string `json:"product_id"`
	Code                string `json:"code"`
	BasePriceCentsMonth int64  `json:"base_price_cents_month"`
	Popular             bool   `json:"popular"`
	Active              bool   `json:"active"`
}

func (d *plansDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plans"
}

func (d *plansDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "All catalog plans available to your account (use a plan `id` when deploying a `palmahost_service`).",
		Attributes: map[string]schema.Attribute{
			"plans": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The available plans.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                     schema.StringAttribute{Computed: true},
						"product_id":             schema.StringAttribute{Computed: true},
						"code":                   schema.StringAttribute{Computed: true},
						"base_price_cents_month": schema.Int64Attribute{Computed: true},
						"popular":                schema.BoolAttribute{Computed: true},
						"active":                 schema.BoolAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *plansDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.c, _ = req.ProviderData.(*client.Client)
}

func (d *plansDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	var plans []apiPlan
	if err := d.c.Do(ctx, "GET", "/plans", nil, &plans); err != nil {
		resp.Diagnostics.AddError("Could not read plans", err.Error())
		return
	}
	out := plansModel{Plans: make([]planModel, 0, len(plans))}
	for _, p := range plans {
		out.Plans = append(out.Plans, planModel{
			ID:                  types.StringValue(p.ID),
			ProductID:           types.StringValue(p.ProductID),
			Code:                types.StringValue(p.Code),
			BasePriceCentsMonth: types.Int64Value(p.BasePriceCentsMonth),
			Popular:             types.BoolValue(p.Popular),
			Active:              types.BoolValue(p.Active),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}
