terraform {
  required_providers {
    palmahost = {
      source = "palmahost/palmahost"
    }
  }
}

# token + base_url come from PALMAHOST_TOKEN / PALMAHOST_BASE_URL env vars.
provider "palmahost" {}

# Read-only: list the catalog plans available to your account.
data "palmahost_plans" "all" {}

output "plan_count" {
  value = length(data.palmahost_plans.all.plans)
}

output "first_plan" {
  value = try(data.palmahost_plans.all.plans[0], null)
}

# Create-and-destroy demo resource (comment out to only read).
resource "palmahost_ssh_key" "trial" {
  name       = "tf-provider-trial"
  public_key = file("~/.ssh/id_ed25519.pub")
}
