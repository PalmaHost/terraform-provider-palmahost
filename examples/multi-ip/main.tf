terraform {
  required_providers {
    palmahost = { source = "palmahost/palmahost" }
  }
}
provider "palmahost" {} # PALMAHOST_TOKEN env; base_url defaults to my.palmahost.sh

resource "palmahost_ssh_key" "deploy" {
  name       = "multi-ip-demo"
  public_key = file("~/.ssh/id_ed25519.pub")
}

# A single VM that comes up with its primary IP plus two extra IPv4 addresses,
# all routed to the same machine. Each extra IP is billed monthly; they are
# attached at deploy and surface in `additional_ips`.
resource "palmahost_compute_vm" "gateway" {
  name       = "gateway"
  image      = "ubuntu-24.04"
  plan       = "pc2-1c-1g"
  location   = "es"
  ssh_key    = palmahost_ssh_key.deploy.id
  extra_ipv4 = 2
}

output "primary_ip"     { value = palmahost_compute_vm.gateway.primary_ip }
output "additional_ips" { value = palmahost_compute_vm.gateway.additional_ips }

# Every address routed to the VM (primary first), e.g. for a reverse proxy that
# binds each site to its own IP.
output "all_ips" {
  value = concat(
    [palmahost_compute_vm.gateway.primary_ip],
    palmahost_compute_vm.gateway.additional_ips,
  )
}
