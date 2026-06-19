package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/palmahost/terraform-provider-palmahost/internal/client"
)

var (
	_ resource.Resource                = (*serviceResource)(nil)
	_ resource.ResourceWithConfigure   = (*serviceResource)(nil)
	_ resource.ResourceWithImportState = (*serviceResource)(nil)
)

// NewServiceResource registers the palmahost_service resource (a deployed VM or
// web-hosting account).
func NewServiceResource() resource.Resource { return &serviceResource{} }

type serviceResource struct{ c *client.Client }

type serviceModel struct {
	ID            types.String `tfsdk:"id"`
	PlanID        types.String `tfsdk:"plan_id"`
	LocationID    types.String `tfsdk:"location_id"`
	Label         types.String `tfsdk:"label"`
	Hostname      types.String `tfsdk:"hostname"`
	OSImage       types.String `tfsdk:"os_image"`
	BillingPeriod types.String `tfsdk:"billing_period"`
	SSHKeyIDs     types.List   `tfsdk:"ssh_key_ids"`
	GroupID       types.String `tfsdk:"group_id"`

	Kind                 types.String `tfsdk:"kind"`
	Status               types.String `tfsdk:"status"`
	UnitPriceLockedCents types.Int64  `tfsdk:"unit_price_locked_cents"`
	Hypervisor           types.String `tfsdk:"hypervisor"`
	ExternalID           types.String `tfsdk:"external_id"`
	PrimaryIP            types.String `tfsdk:"primary_ip"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
}

type apiInstance struct {
	ID                   string `json:"id"`
	PlanID               string `json:"plan_id"`
	LocationID           string `json:"location_id"`
	Kind                 string `json:"kind"`
	Label                string `json:"label"`
	Hostname             string `json:"hostname"`
	Status               string `json:"status"`
	BillingPeriod        string `json:"billing_period"`
	UnitPriceLockedCents int64  `json:"unit_price_locked_cents"`
	Hypervisor           string `json:"hypervisor"`
	ExternalID           string `json:"external_id"`
	OSImage              string `json:"os_image"`
	PrimaryIP            string `json:"primary_ip"`
	CreatedAt            string `json:"created_at"`
	UpdatedAt            string `json:"updated_at"`
}

func (r *serviceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *serviceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replaceStr := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		MarkdownDescription: "A deployed service: a VPS or a web-hosting account. Creating one deploys it and waits until it is active; destroying it terminates it.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, MarkdownDescription: "Service identifier."},
			"plan_id":     schema.StringAttribute{Required: true, PlanModifiers: replaceStr, MarkdownDescription: "Plan id (see the `palmahost_plans` data source)."},
			"location_id": schema.StringAttribute{Required: true, PlanModifiers: replaceStr, MarkdownDescription: "Location id (see the `palmahost_locations` data source)."},
			"label":       schema.StringAttribute{Required: true, MarkdownDescription: "Display name (editable in place)."},
			"hostname":    schema.StringAttribute{Required: true, PlanModifiers: replaceStr, MarkdownDescription: "Hostname."},
			"os_image":    schema.StringAttribute{Required: true, PlanModifiers: replaceStr, MarkdownDescription: "OS image, e.g. `ubuntu-24.04` (for web hosting use the kind's accepted value)."},
			"billing_period": schema.StringAttribute{Required: true, PlanModifiers: replaceStr, MarkdownDescription: "`monthly`, `annual` or `hourly`."},
			"ssh_key_ids": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				PlanModifiers:       []planmodifier.List{listplanmodifier.RequiresReplace()},
				MarkdownDescription: "SSH key ids to install at deploy time (see `palmahost_ssh_key`).",
			},
			"group_id": schema.StringAttribute{Optional: true, PlanModifiers: replaceStr, MarkdownDescription: "Optional project/group id."},

			"kind":                    schema.StringAttribute{Computed: true, MarkdownDescription: "`vps` | `webhosting` | …"},
			"status":                  schema.StringAttribute{Computed: true, MarkdownDescription: "Lifecycle status (active, stopped, …)."},
			"unit_price_locked_cents": schema.Int64Attribute{Computed: true, MarkdownDescription: "Locked recurring price in cents."},
			"hypervisor":              schema.StringAttribute{Computed: true},
			"external_id":             schema.StringAttribute{Computed: true},
			"primary_ip":              schema.StringAttribute{Computed: true, MarkdownDescription: "Primary IPv4 address (once provisioned)."},
			"created_at":              schema.StringAttribute{Computed: true},
			"updated_at":              schema.StringAttribute{Computed: true},
		},
	}
}

func (r *serviceResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.c, _ = req.ProviderData.(*client.Client)
}

func (r *serviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m serviceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"plan_id":        m.PlanID.ValueString(),
		"location_id":    m.LocationID.ValueString(),
		"label":          m.Label.ValueString(),
		"hostname":       m.Hostname.ValueString(),
		"os_image":       m.OSImage.ValueString(),
		"billing_period": m.BillingPeriod.ValueString(),
	}
	if !m.SSHKeyIDs.IsNull() && !m.SSHKeyIDs.IsUnknown() {
		var ids []string
		resp.Diagnostics.Append(m.SSHKeyIDs.ElementsAs(ctx, &ids, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		body["ssh_key_ids"] = ids
	}
	if v := m.GroupID.ValueString(); v != "" {
		body["group_id"] = v
	}

	var out apiInstance
	if err := r.c.Do(ctx, "POST", "/services", body, &out); err != nil {
		resp.Diagnostics.AddError("Could not deploy service", err.Error())
		return
	}

	// Wait until the deploy finishes (or surface a warning so the resource is
	// still tracked and can be inspected/destroyed — never orphaned).
	inst, waitErr := pollServiceReady(ctx, r.c, out.ID, out)
	applyInstance(&m, inst)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
	if waitErr != nil {
		resp.Diagnostics.AddWarning("Service not ready yet", waitErr.Error()+" — run `terraform apply` again to refresh once it settles.")
	}
}

func (r *serviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m serviceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var inst apiInstance
	if err := r.c.Do(ctx, "GET", "/services/"+m.ID.ValueString()+"/", nil, &inst); err != nil {
		if ae, ok := err.(*client.APIError); ok && ae.Status == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Could not read service", err.Error())
		return
	}
	if inst.Status == "terminated" {
		resp.State.RemoveResource(ctx)
		return
	}
	applyInstance(&m, inst) // ssh_key_ids is input-only (not returned); left as-is
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *serviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state serviceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Only `label` is editable in place (everything else requires replacement).
	if plan.Label.ValueString() != state.Label.ValueString() {
		var inst apiInstance
		if err := r.c.Do(ctx, "PATCH", "/services/"+state.ID.ValueString()+"/", map[string]string{"label": plan.Label.ValueString()}, &inst); err != nil {
			resp.Diagnostics.AddError("Could not update service label", err.Error())
			return
		}
		applyInstance(&plan, inst)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m serviceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.c.Do(ctx, "DELETE", "/services/"+m.ID.ValueString()+"/", nil, nil); err != nil {
		if ae, ok := err.(*client.APIError); ok && ae.Status == 404 {
			return
		}
		resp.Diagnostics.AddError("Could not terminate service", err.Error())
	}
}

func (r *serviceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// pollServiceReady polls the service until it is active/stopped, errors, or
// times out. On a non-ready outcome it returns the latest instance plus a
// descriptive error (callers turn that into a warning, not a hard failure).
func pollServiceReady(ctx context.Context, c *client.Client, id string, last apiInstance) (apiInstance, error) {
	deadline := time.Now().Add(15 * time.Minute)
	for {
		var inst apiInstance
		if err := c.Do(ctx, "GET", "/services/"+id+"/", nil, &inst); err != nil {
			return last, err
		}
		last = inst
		switch inst.Status {
		case "active", "stopped":
			return inst, nil
		case "error":
			return inst, fmt.Errorf("the service entered the %q state during provisioning", inst.Status)
		}
		if time.Now().After(deadline) {
			return inst, fmt.Errorf("timed out after 15m waiting for the service to become active (status %q)", inst.Status)
		}
		select {
		case <-ctx.Done():
			return last, ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

func applyInstance(m *serviceModel, i apiInstance) {
	m.ID = types.StringValue(i.ID)
	m.PlanID = types.StringValue(i.PlanID)
	m.LocationID = types.StringValue(i.LocationID)
	m.Label = types.StringValue(i.Label)
	m.Hostname = types.StringValue(i.Hostname)
	m.OSImage = types.StringValue(i.OSImage)
	m.BillingPeriod = types.StringValue(i.BillingPeriod)
	m.Kind = types.StringValue(i.Kind)
	m.Status = types.StringValue(i.Status)
	m.UnitPriceLockedCents = types.Int64Value(i.UnitPriceLockedCents)
	m.Hypervisor = types.StringValue(i.Hypervisor)
	m.ExternalID = types.StringValue(i.ExternalID)
	m.PrimaryIP = types.StringValue(i.PrimaryIP)
	m.CreatedAt = types.StringValue(i.CreatedAt)
	m.UpdatedAt = types.StringValue(i.UpdatedAt)
}
