package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/palmahost/terraform-provider-palmahost/internal/client"
)

var (
	_ resource.Resource                = (*vmResource)(nil)
	_ resource.ResourceWithConfigure   = (*vmResource)(nil)
	_ resource.ResourceWithImportState = (*vmResource)(nil)
)

// NewComputeVMResource registers palmahost_compute_vm — an ergonomic VM resource
// that takes human-friendly plan/location codes and a single SSH key.
func NewComputeVMResource() resource.Resource { return &vmResource{} }

type vmResource struct{ c *client.Client }

type vmModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Image         types.String `tfsdk:"image"`
	Plan          types.String `tfsdk:"plan"`
	Location      types.String `tfsdk:"location"`
	SSHKey        types.String `tfsdk:"ssh_key"`
	BillingPeriod types.String `tfsdk:"billing_period"`
	ExtraIPv4     types.Int64  `tfsdk:"extra_ipv4"`

	PlanID        types.String `tfsdk:"plan_id"`
	LocationID    types.String `tfsdk:"location_id"`
	Status        types.String `tfsdk:"status"`
	PrimaryIP     types.String `tfsdk:"primary_ip"`
	AdditionalIPs types.List   `tfsdk:"additional_ips"`
	PriceCents    types.Int64  `tfsdk:"price_cents"`
	CreatedAt     types.String `tfsdk:"created_at"`
}

func (r *vmResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_compute_vm"
}

func (r *vmResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Deploy a virtual machine. `plan` and `location` accept human codes (e.g. `pc2-1c-1g`, `es` or `mad`); the provider resolves them to ids. Creating waits until the VM is active; destroying terminates it.",
		Attributes: map[string]schema.Attribute{
			"id":       schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name":     schema.StringAttribute{Required: true, PlanModifiers: replace, MarkdownDescription: "VM name (used as the label and hostname)."},
			"image":    schema.StringAttribute{Required: true, PlanModifiers: replace, MarkdownDescription: "OS image, e.g. `ubuntu-24.04`."},
			"plan":     schema.StringAttribute{Required: true, PlanModifiers: replace, MarkdownDescription: "Plan code, e.g. `pc2-1c-1g`."},
			"location": schema.StringAttribute{Required: true, PlanModifiers: replace, MarkdownDescription: "Location code (`mad`) or country (`es`)."},
			"ssh_key":  schema.StringAttribute{Optional: true, PlanModifiers: replace, MarkdownDescription: "SSH key id to install (see `palmahost_ssh_key`)."},
			"billing_period": schema.StringAttribute{
				Optional: true, Computed: true,
				Default:             stringdefault.StaticString("hourly"),
				PlanModifiers:       replace,
				MarkdownDescription: "`hourly` (default — pay as you go), `monthly` or `annual`.",
			},
			"extra_ipv4": schema.Int64Attribute{
				Optional: true, Computed: true,
				Default:             int64default.StaticInt64(0),
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
				MarkdownDescription: "Number of additional IPv4 addresses to allocate and attach at deploy (each billed monthly). They appear in `additional_ips`.",
			},

			"plan_id":        schema.StringAttribute{Computed: true, MarkdownDescription: "Resolved plan id."},
			"location_id":    schema.StringAttribute{Computed: true, MarkdownDescription: "Resolved location id."},
			"status":         schema.StringAttribute{Computed: true},
			"primary_ip":     schema.StringAttribute{Computed: true, MarkdownDescription: "Primary IPv4 (once provisioned)."},
			"additional_ips": schema.ListAttribute{Computed: true, ElementType: types.StringType, MarkdownDescription: "Additional IPv4 addresses attached to this VM (from `extra_ipv4`)."},
			"price_cents":    schema.Int64Attribute{Computed: true, MarkdownDescription: "Locked recurring price in cents."},
			"created_at":     schema.StringAttribute{Computed: true},
		},
	}
}

func (r *vmResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.c, _ = req.ProviderData.(*client.Client)
}

func (r *vmResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m vmModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}

	planID, err := resolvePlanCode(ctx, r.c, m.Plan.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unknown plan", err.Error())
		return
	}
	locID, err := resolveLocation(ctx, r.c, m.Location.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unknown location", err.Error())
		return
	}

	body := map[string]any{
		"plan_id":        planID,
		"location_id":    locID,
		"label":          m.Name.ValueString(),
		"hostname":       m.Name.ValueString(),
		"os_image":       m.Image.ValueString(),
		"billing_period": m.BillingPeriod.ValueString(),
	}
	if v := m.SSHKey.ValueString(); v != "" {
		body["ssh_key_ids"] = []string{v}
	}
	if n := m.ExtraIPv4.ValueInt64(); n > 0 {
		body["config"] = map[string]any{"extra_ipv4": n}
	}

	var out apiInstance
	if err := r.c.Do(ctx, "POST", "/services", body, &out); err != nil {
		resp.Diagnostics.AddError("Could not deploy VM", err.Error())
		return
	}
	inst, waitErr := pollServiceReady(ctx, r.c, out.ID, out)
	applyVM(&m, inst)
	resp.Diagnostics.Append(setAdditionalIPs(ctx, r.c, &m)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
	if waitErr != nil {
		resp.Diagnostics.AddWarning("VM not ready yet", waitErr.Error()+" — re-run `terraform apply` to refresh once it settles.")
	}
}

func (r *vmResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m vmModel
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
		resp.Diagnostics.AddError("Could not read VM", err.Error())
		return
	}
	if inst.Status == "terminated" {
		resp.State.RemoveResource(ctx)
		return
	}
	applyVM(&m, inst) // keeps the user-supplied codes (plan/location/name/image/ssh_key)
	resp.Diagnostics.Append(setAdditionalIPs(ctx, r.c, &m)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *vmResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Every editable attribute requires replacement; nothing updates in place.
	var m vmModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *vmResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m vmModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.c.Do(ctx, "DELETE", "/services/"+m.ID.ValueString()+"/", nil, nil); err != nil {
		if ae, ok := err.(*client.APIError); ok && ae.Status == 404 {
			return
		}
		resp.Diagnostics.AddError("Could not terminate VM", err.Error())
	}
}

func (r *vmResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// applyVM copies the server-computed fields from the API instance, leaving the
// user-supplied inputs (name/image/plan/location/ssh_key/billing_period) intact.
func applyVM(m *vmModel, i apiInstance) {
	m.ID = types.StringValue(i.ID)
	m.PlanID = types.StringValue(i.PlanID)
	m.LocationID = types.StringValue(i.LocationID)
	m.Status = types.StringValue(i.Status)
	m.PrimaryIP = types.StringValue(i.PrimaryIP)
	m.PriceCents = types.Int64Value(i.UnitPriceLockedCents)
	m.CreatedAt = types.StringValue(i.CreatedAt)
}

// apiOwnedIP is the subset of GET /v1/ips needed to list a VM's attached IPs.
type apiOwnedIP struct {
	Address           string `json:"address"`
	IsPrimary         bool   `json:"is_primary"`
	AttachedServiceID string `json:"attached_service_id"`
}

// setAdditionalIPs fills m.AdditionalIPs with the IPv4 addresses routed to this
// VM beyond its primary (GET /v1/ips, filtered by attachment). Addresses are
// normalized (the /32 host mask is stripped) and the one already reported as
// primary_ip is excluded, so a single address never appears twice. The list
// stays known (empty) even if the lookup fails, so the attribute never blocks.
func setAdditionalIPs(ctx context.Context, c *client.Client, m *vmModel) diag.Diagnostics {
	addrs := []string{}
	var ips []apiOwnedIP
	if err := c.Do(ctx, "GET", "/ips", nil, &ips); err == nil {
		id := m.ID.ValueString()
		primary := m.PrimaryIP.ValueString()
		seen := map[string]bool{}
		for _, ip := range ips {
			if ip.IsPrimary || ip.AttachedServiceID != id {
				continue
			}
			addr, _, _ := strings.Cut(ip.Address, "/") // drop the /32 host mask
			if addr == "" || addr == primary || seen[addr] {
				continue
			}
			seen[addr] = true
			addrs = append(addrs, addr)
		}
		sort.Strings(addrs)
	}
	list, d := types.ListValueFrom(ctx, types.StringType, addrs)
	m.AdditionalIPs = list
	return d
}

// resolvePlanCode maps a plan code (e.g. "pc2-1c-1g") to its id.
func resolvePlanCode(ctx context.Context, c *client.Client, code string) (string, error) {
	var plans []apiPlan
	if err := c.Do(ctx, "GET", "/plans", nil, &plans); err != nil {
		return "", err
	}
	for _, p := range plans {
		if p.Code == code {
			return p.ID, nil
		}
	}
	return "", fmt.Errorf("no plan found with code %q", code)
}

// resolveLocation maps a location code ("mad") or country ("es") to a location id.
func resolveLocation(ctx context.Context, c *client.Client, val string) (string, error) {
	var locs []apiLocation
	if err := c.Do(ctx, "GET", "/locations", nil, &locs); err != nil {
		return "", err
	}
	v := strings.ToLower(strings.TrimSpace(val))
	for _, l := range locs {
		if strings.ToLower(l.Code) == v {
			return l.ID, nil
		}
	}
	for _, l := range locs {
		if l.Active && strings.ToLower(l.Country) == v {
			return l.ID, nil
		}
	}
	return "", fmt.Errorf("no location matching %q (try a location code like \"mad\" or a country like \"es\")", val)
}
