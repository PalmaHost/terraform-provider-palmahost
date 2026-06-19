---
page_title: "palmahost_service Resource - palmahost"
description: |-
  Lower-level deploy of any service kind by explicit plan and location ids.
---

# palmahost_service (Resource)

The lower-level deploy primitive. Use it for web hosting, or when you want to
pass explicit ids instead of the friendly codes that `palmahost_compute_vm`
resolves. `label` is editable in place; the other inputs are immutable.

## Example Usage

```terraform
data "palmahost_plans" "all" {}
data "palmahost_locations" "all" {}

locals {
  plan = [for p in data.palmahost_plans.all.plans : p if p.code == "pc2-1c-1g"][0]
  loc  = [for l in data.palmahost_locations.all.locations : l if l.code == "mad"][0]
}

resource "palmahost_service" "vm" {
  plan_id        = local.plan.id
  location_id    = local.loc.id
  label          = "app-server"
  hostname       = "app-1"
  os_image       = "ubuntu-24.04"
  billing_period = "monthly"
}
```

## Schema

### Required

- `plan_id` (String) Plan id (see the `palmahost_plans` data source).
- `location_id` (String) Location id (see the `palmahost_locations` data source).
- `label` (String) Display name (editable in place).
- `hostname` (String) Hostname.
- `os_image` (String) OS image, e.g. `ubuntu-24.04` (for web hosting, the kind's
  accepted value).
- `billing_period` (String) `monthly`, `annual`, or `hourly`.

### Optional

- `ssh_key_ids` (List of String) SSH key ids to install.
- `group_id` (String) Optional project/group id.

### Read-Only

- `id` (String) Service id.
- `kind` (String) `vps` | `webhosting` | …
- `status` (String) Lifecycle status (active, stopped, …).
- `unit_price_locked_cents` (Number) Locked recurring price in cents.
- `hypervisor` (String)
- `external_id` (String)
- `primary_ip` (String) Primary IPv4 address (once provisioned).
- `created_at` (String)
- `updated_at` (String)
