---
page_title: "Provider: PalmaHost Cloud"
description: |-
  Manage PalmaHost Cloud resources (VMs, web hosting, SSH keys) with Terraform.
---

# PalmaHost Cloud Provider

Manage [PalmaHost Cloud](https://palmahost.sh) resources declaratively. The
provider talks to the public REST API at `https://my.palmahost.sh/v1`.

## Example Usage

```terraform
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
}

resource "palmahost_compute_vm" "web" {
  name     = "web-1"
  image    = "ubuntu-24.04"
  plan     = "pc2-1c-1g"
  location = "es"
}
```

## Authentication

Create an API token in the panel under **Account → API tokens** and scope it to
what your configuration manages (e.g. `services:write`, `ssh-keys:write`). Pass
it as the `token` argument or the `PALMAHOST_TOKEN` environment variable.

## Schema

### Optional

- `token` (String, Sensitive) API token (`ph_live_…`). May also be set with the
  `PALMAHOST_TOKEN` environment variable.
- `base_url` (String) API base URL including the version prefix. Defaults to
  `https://my.palmahost.sh/v1`. May also be set with `PALMAHOST_BASE_URL` (use
  `https://my-dev.palmahost.sh/v1` for the dev instance).
