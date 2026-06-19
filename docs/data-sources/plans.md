---
page_title: "palmahost_plans Data Source - palmahost"
description: |-
  The catalog of available plans.
---

# palmahost_plans (Data Source)

Lists the catalog plans available to your account, with their codes and pricing.

## Example Usage

```terraform
data "palmahost_plans" "all" {}

output "vps_plans" {
  value = [for p in data.palmahost_plans.all.plans : p.code if p.active]
}
```

## Schema

### Read-Only

- `plans` (List of Object) Each element has:
  - `id` (String)
  - `product_id` (String)
  - `code` (String) The friendly code, e.g. `pc2-1c-1g`.
  - `base_price_cents_month` (Number)
  - `popular` (Boolean)
  - `active` (Boolean)
