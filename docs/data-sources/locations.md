---
page_title: "palmahost_locations Data Source - palmahost"
description: |-
  The available deployment regions.
---

# palmahost_locations (Data Source)

Lists the deployment regions (locations) available to your account.

## Example Usage

```terraform
data "palmahost_locations" "all" {}

output "regions" {
  value = [for l in data.palmahost_locations.all.locations : l.code if l.active]
}
```

## Schema

### Read-Only

- `locations` (List of Object) Each element has:
  - `id` (String)
  - `code` (String) The region code, e.g. `mad`.
  - `name` (String)
  - `country` (String) ISO country code, e.g. `es`.
  - `active` (Boolean)
