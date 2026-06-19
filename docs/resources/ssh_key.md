---
page_title: "palmahost_ssh_key Resource - palmahost"
description: |-
  An SSH public key that can be installed on VMs at deploy.
---

# palmahost_ssh_key (Resource)

Registers an SSH public key on your account. Keys are immutable: changing `name`
or `public_key` replaces the key.

## Example Usage

```terraform
resource "palmahost_ssh_key" "deploy" {
  name       = "ci-deploy"
  public_key = file("~/.ssh/id_ed25519.pub")
}
```

## Schema

### Required

- `name` (String) Display name for the key.
- `public_key` (String) The SSH public key material (e.g. the contents of
  `id_ed25519.pub`).

### Read-Only

- `id` (String) Key id.
- `fingerprint` (String) Key fingerprint.
- `created_at` (String)

## Import

```shell
terraform import palmahost_ssh_key.deploy ssh_019e...
```
