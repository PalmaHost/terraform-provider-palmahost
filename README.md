# Terraform Provider for PalmaHost Cloud

Manage [PalmaHost Cloud](https://palmahost.sh) resources declaratively. The
provider talks to the same public REST API as the SDKs
(`https://my.palmahost.sh/v1`) and is generated against its OpenAPI spec, so it
stays in sync with the platform.

- **Source:** `palmahost/palmahost`
- **Resources:** `palmahost_compute_vm`, `palmahost_service`, `palmahost_ssh_key`
- **Data sources:** `palmahost_plans`, `palmahost_locations`

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.0
  (or [OpenTofu](https://opentofu.org) >= 1.6)
- A PalmaHost API token (`ph_live_…`) — create one in the panel under
  **Account → API tokens** and scope it to what your config manages
  (e.g. `services:write`, `ssh-keys:write`, `network:write`).

## Install

Once published, declare the provider and run `terraform init`:

```hcl
terraform {
  required_providers {
    palmahost = {
      source  = "palmahost/palmahost"
      version = "~> 0.1"
    }
  }
}

provider "palmahost" {
  token = var.palmahost_token # or the PALMAHOST_TOKEN env var
  # base_url = "https://my.palmahost.sh/v1" # default; override per environment
}
```

To run it **before** it is on the registry, use Terraform `dev_overrides` (no
`terraform init`, build the binary locally) — see [TRIAL.md](./TRIAL.md).

## Configuration

| Argument   | Env                  | Default                       | Notes                      |
|------------|----------------------|-------------------------------|----------------------------|
| `token`    | `PALMAHOST_TOKEN`    | —                             | API token (`ph_live_…`)    |
| `base_url` | `PALMAHOST_BASE_URL` | `https://my.palmahost.sh/v1`  | API base URL incl. version |

The panel and API share the host, so there is no separate `api.*` subdomain.
Point `base_url` at `https://my-dev.palmahost.sh/v1` for the dev instance.

## Resources & data sources

| Type | Name | Purpose |
|------|------|---------|
| resource | `palmahost_compute_vm` | Deploy a VM from human codes (`plan = "pc2-1c-1g"`, `location = "es"`/`"mad"`, single `ssh_key`, optional `extra_ipv4`). Waits until active; terminates on destroy. |
| resource | `palmahost_service`    | Lower-level deploy (any kind, incl. web hosting) by explicit `plan_id` / `location_id` / `label` / `hostname` / `os_image` / `billing_period`. |
| resource | `palmahost_ssh_key`    | An SSH public key (create / delete / import; immutable). |
| data | `palmahost_plans`       | Catalog plans (`code`, `product_id`, price, …). |
| data | `palmahost_locations`   | Deployment regions (`code`, `country`, `active`). |

More resources (`palmahost_ip`, `palmahost_firewall_rule`, `palmahost_cron_job`,
`palmahost_webhook`) are on the roadmap.

---

## Examples

Runnable copies of each live in [`examples/`](./examples). Set your token once:

```bash
export PALMAHOST_TOKEN="ph_live_…"
```

### 1. Deploy a single VM

[`examples/compute-vm`](./examples/compute-vm) — `plan` and `location` take
friendly codes; the provider resolves them to ids and waits until the VM is
`active`.

```hcl
resource "palmahost_ssh_key" "deploy" {
  name       = "ci-deploy"
  public_key = file("~/.ssh/id_ed25519.pub")
}

resource "palmahost_compute_vm" "web" {
  name     = "web-1"
  image    = "ubuntu-24.04"
  plan     = "pc2-1c-1g"
  location = "es"
  ssh_key  = palmahost_ssh_key.deploy.id
  # billing_period = "hourly" # default; or "monthly" / "annual"
}

output "ip" { value = palmahost_compute_vm.web.primary_ip }
```

```bash
terraform apply      # deploys, waits until active, prints the IP
terraform destroy    # terminates it
```

### 2. A fleet of VMs, run a command on each, then tear it down

[`examples/fleet`](./examples/fleet) — `count` stamps out N identical VMs; a
`remote-exec` provisioner runs a command on each as it boots and streams the
output into the apply log. Uses Terraform core only (works in the dev trial).

```hcl
variable "fleet_size" { default = 4 }

resource "palmahost_compute_vm" "node" {
  count    = var.fleet_size
  name     = "node-${count.index + 1}"
  image    = "ubuntu-24.04"
  plan     = "pc2-1c-1g"
  location = "es"
  ssh_key  = palmahost_ssh_key.deploy.id

  provisioner "remote-exec" {
    inline = [
      "echo \"[$(hostname)] kernel $(uname -r) | $(nproc) vCPU | $(free -m | awk '/Mem:/{print $2}') MB RAM\"",
    ]
    connection {
      type        = "ssh"
      host        = self.primary_ip
      user        = "root"
      private_key = file("~/.ssh/id_ed25519")
      timeout     = "3m"
    }
  }
}

output "fleet" {
  value = { for vm in palmahost_compute_vm.node : vm.name => vm.primary_ip }
}
```

```bash
terraform apply                       # deploys 4 VMs, runs the command on each
terraform output fleet                # { "node-1" = "…", … }
terraform destroy                     # terminates all 4 in one step
```

### 3. Run a command on each and capture the result into an output

[`examples/run-command`](./examples/run-command) — when you need the command's
output as a *value* (not just in the log), an `external` data source SSHes into
each node and returns its stdout. Requires `terraform init` (the
`hashicorp/external` provider) plus `jq` and `ssh` locally.

```hcl
data "external" "run" {
  count = var.fleet_size
  program = ["bash", "-c", <<-EOT
    out=$(ssh -o StrictHostKeyChecking=no -i ~/.ssh/id_ed25519 \
      root@${palmahost_compute_vm.node[count.index].primary_ip} ${jsonencode(var.command)} 2>&1)
    jq -n --arg result "$out" '{"result": $result}'
  EOT
  ]
}

output "results" {
  value = { for i, vm in palmahost_compute_vm.node :
    vm.name => trimspace(data.external.run[i].result["result"]) }
}
```

```bash
terraform apply -var 'command=uptime -p'
terraform output results   # { "node-1" = "up 2 minutes", … }
```

### 4. A VM with multiple IP addresses

[`examples/multi-ip`](./examples/multi-ip) — `extra_ipv4` allocates and attaches
additional IPv4 addresses at deploy (each billed monthly). They surface in the
computed `additional_ips` list.

```hcl
resource "palmahost_compute_vm" "gateway" {
  name       = "gateway"
  image      = "ubuntu-24.04"
  plan       = "pc2-1c-1g"
  location   = "es"
  ssh_key    = palmahost_ssh_key.deploy.id
  extra_ipv4 = 2
}

output "all_ips" {
  value = concat([palmahost_compute_vm.gateway.primary_ip],
    palmahost_compute_vm.gateway.additional_ips)
}
```

### 5. Look up plans and locations

```hcl
data "palmahost_plans" "all" {}
data "palmahost_locations" "all" {}

output "vps_plans" {
  value = [for p in data.palmahost_plans.all.plans : p.code if p.active]
}
output "regions" {
  value = [for l in data.palmahost_locations.all.locations : l.code if l.active]
}
```

### 6. Import an existing resource

```bash
terraform import palmahost_ssh_key.deploy ssh_019e…
terraform import palmahost_compute_vm.web srv_019e…
```

---

## `palmahost_compute_vm` attributes

| Attribute | | Description |
|-----------|---|-------------|
| `name` | required | VM name (used as label and hostname). |
| `image` | required | OS image, e.g. `ubuntu-24.04`. |
| `plan` | required | Plan code, e.g. `pc2-1c-1g`. |
| `location` | required | Location code (`mad`) or country (`es`). |
| `ssh_key` | optional | SSH key id to install (see `palmahost_ssh_key`). |
| `billing_period` | optional | `hourly` (default), `monthly`, or `annual`. |
| `extra_ipv4` | optional | Additional IPv4 addresses to attach at deploy (default `0`). |
| `id`, `plan_id`, `location_id`, `status`, `primary_ip`, `additional_ips`, `price_cents`, `created_at` | computed | Server-assigned. |

Changing any input replaces the VM (deploy is immutable: destroy + create).

## Development

```bash
go build -o terraform-provider-palmahost .   # build
go test ./...                                # unit tests
```

## License

[MPL-2.0](./LICENSE).
