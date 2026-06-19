# Trying the provider locally (before publishing)

Terraform's `dev_overrides` lets you run an unpublished provider straight from a
local binary — no registry, no `terraform init`. This is verified working against
the dev instance (`https://my.palmahost.sh/v1`).

## 1. Build the provider binary

You need Go ≥ 1.23 (or ask me for a prebuilt binary for your OS/arch).

```bash
cd terraform-provider-palmahost
go build -o terraform-provider-palmahost .
pwd   # note this absolute path — it's the dev_overrides directory
```

## 2. Point Terraform at the local binary

Create `~/.terraformrc` (or set `TF_CLI_CONFIG_FILE` to a file with):

```hcl
provider_installation {
  dev_overrides {
    "palmahost/palmahost" = "/ABSOLUTE/PATH/TO/terraform-provider-palmahost"
  }
  direct {}
}
```

## 3. Configure + run

```bash
export PALMAHOST_TOKEN="ph_live_…"                 # Account → API tokens
export PALMAHOST_BASE_URL="https://my.palmahost.sh/v1"   # dev instance
cd examples/trial
terraform plan      # do NOT run `terraform init` — see below
terraform apply
terraform destroy
```

> ⚠️ **Do not run `terraform init`.** With `dev_overrides`, Terraform uses the
> local binary directly and skips init. Running `init` makes Terraform try to
> fetch `palmahost/palmahost` from the public registry (where it isn't published
> yet) and fail with "provider registry … does not have a provider named …". If
> you see an "inconsistent dependency lock file" error from a previous `init`,
> delete `.terraform.lock.hcl` and the `.terraform/` directory and just run
> `terraform plan`.

`dev_overrides` prints a warning ("development overrides are in effect") — that's
expected. When the provider is published to the Terraform Registry, drop the
override and use a normal `terraform init`.
