package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/palmahost/terraform-provider-palmahost/internal/client"
)

var (
	_ resource.Resource                = (*sshKeyResource)(nil)
	_ resource.ResourceWithConfigure   = (*sshKeyResource)(nil)
	_ resource.ResourceWithImportState = (*sshKeyResource)(nil)
)

// NewSSHKeyResource registers the palmahost_ssh_key resource.
func NewSSHKeyResource() resource.Resource { return &sshKeyResource{} }

type sshKeyResource struct{ c *client.Client }

type sshKeyModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	PublicKey   types.String `tfsdk:"public_key"`
	Fingerprint types.String `tfsdk:"fingerprint"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

type apiKey struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	PublicKey   string `json:"public_key"`
	Fingerprint string `json:"fingerprint"`
	CreatedAt   string `json:"created_at"`
}

func (r *sshKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_key"
}

func (r *sshKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "An SSH public key on your account, selectable when deploying servers. Keys are immutable; changing `name` or `public_key` replaces the key.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Key identifier.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Human-readable name.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"public_key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The OpenSSH public key (e.g. `ssh-ed25519 AAAA…`).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"fingerprint": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Server-computed fingerprint.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Creation timestamp (RFC 3339).",
			},
		},
	}
}

func (r *sshKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.c, _ = req.ProviderData.(*client.Client)
}

func (r *sshKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m sshKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var out apiKey
	body := map[string]string{"name": m.Name.ValueString(), "public_key": m.PublicKey.ValueString()}
	if err := r.c.Do(ctx, "POST", "/ssh-keys/", body, &out); err != nil {
		resp.Diagnostics.AddError("Could not create SSH key", err.Error())
		return
	}
	applyKey(&m, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *sshKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m sshKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// The API has no single-key GET; list and match by id.
	var keys []apiKey
	if err := r.c.Do(ctx, "GET", "/ssh-keys/", nil, &keys); err != nil {
		resp.Diagnostics.AddError("Could not read SSH keys", err.Error())
		return
	}
	for _, k := range keys {
		if k.ID == m.ID.ValueString() {
			applyKey(&m, k)
			resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
			return
		}
	}
	resp.State.RemoveResource(ctx) // deleted out of band
}

// Update only ever runs for in-place changes; every editable attribute requires
// replacement, so this just persists the (unchanged) plan.
func (r *sshKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var m sshKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *sshKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m sshKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.c.Do(ctx, "DELETE", "/ssh-keys/"+m.ID.ValueString(), nil, nil); err != nil {
		resp.Diagnostics.AddError("Could not delete SSH key", err.Error())
	}
}

func (r *sshKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyKey(m *sshKeyModel, k apiKey) {
	m.ID = types.StringValue(k.ID)
	m.Name = types.StringValue(k.Name)
	m.PublicKey = types.StringValue(k.PublicKey)
	m.Fingerprint = types.StringValue(k.Fingerprint)
	m.CreatedAt = types.StringValue(k.CreatedAt)
}
