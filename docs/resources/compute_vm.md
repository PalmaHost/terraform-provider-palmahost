---
page_title: "palmahost_compute_vm Resource - palmahost"
description: |-
  Deploy a virtual machine from human-friendly plan and location codes.
---

# palmahost_compute_vm (Resource)

Deploy a virtual machine. `plan` and `location` accept human codes (e.g.
`pc2-1c-1g`, `es` or `mad`) and the provider resolves them to ids. Creating
waits until the VM is `active`; destroying terminates it. Every input is
immutable: changing one replaces the VM.

## Example Usage

```terraform
resource "palmahost_ssh_key" "deploy" {
  name       = "ci-deploy"
  public_key = file("~/.ssh/id_ed25519.pub")
}

resource "palmahost_compute_vm" "web" {
  name       = "web-1"
  image      = "ubuntu-24.04"
  plan       = "pc2-1c-1g"
  location   = "es"
  ssh_key    = palmahost_ssh_key.deploy.id
  extra_ipv4 = 1 # one additional IPv4, attached at deploy
}

output "ip" { value = palmahost_compute_vm.web.primary_ip }
```

## Schema

### Required

- `name` (String) VM name (used as the label and hostname).
- `image` (String) OS image, e.g. `ubuntu-24.04`.
- `plan` (String) Plan code, e.g. `pc2-1c-1g`.
- `location` (String) Location code (`mad`) or country (`es`).

### Optional

- `ssh_key` (String) SSH key id to install (see `palmahost_ssh_key`).
- `billing_period` (String) `hourly` (default, pay as you go), `monthly`, or
  `annual`.
- `extra_ipv4` (Number) Number of additional IPv4 addresses to allocate and
  attach at deploy (each billed monthly). Defaults to `0`. They appear in
  `additional_ips`.

### Read-Only

- `id` (String) Service id.
- `plan_id` (String) Resolved plan id.
- `location_id` (String) Resolved location id.
- `status` (String) Lifecycle status.
- `primary_ip` (String) Primary IPv4 address.
- `additional_ips` (List of String) Additional IPv4 addresses attached to this
  VM.
- `price_cents` (Number) Locked recurring price in cents.
- `created_at` (String)

## Import

```shell
terraform import palmahost_compute_vm.web srv_019e...
```
