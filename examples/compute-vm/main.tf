terraform {
  required_providers {
    palmahost = { source = "palmahost/palmahost" }
  }
}
provider "palmahost" {} # PALMAHOST_TOKEN env; base_url defaults to my.palmahost.sh

resource "palmahost_ssh_key" "ssh_key" {
  name       = "test-ssh-key"
  public_key = file("~/.ssh/id_ed25519.pub")
}

resource "palmahost_compute_vm" "test" {
  name     = "test-vm-terraform"
  image    = "ubuntu-24.04"
  plan     = "pc2-1c-1g"
  location = "es"
  ssh_key  = palmahost_ssh_key.ssh_key.id
}

output "vm_status" { value = palmahost_compute_vm.test.status }
output "vm_ip"     { value = palmahost_compute_vm.test.primary_ip }
