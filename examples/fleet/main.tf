terraform {
  required_providers {
    palmahost = { source = "palmahost/palmahost" }
  }
}
provider "palmahost" {} # PALMAHOST_TOKEN env; base_url defaults to my.palmahost.sh

variable "fleet_size" {
  type        = number
  default     = 4
  description = "How many identical VMs to deploy."
}

resource "palmahost_ssh_key" "deploy" {
  name       = "fleet-demo"
  public_key = file("~/.ssh/id_ed25519.pub")
}

# A fleet of identical VMs. `count` stamps out fleet_size copies, each with a
# unique name. The apply waits until every VM is active before returning.
resource "palmahost_compute_vm" "node" {
  count    = var.fleet_size
  name     = "node-${count.index + 1}"
  image    = "ubuntu-24.04"
  plan     = "pc2-1c-1g"
  location = "es"
  ssh_key  = palmahost_ssh_key.deploy.id

  # Run a command on each node as soon as it boots. `remote-exec` streams the
  # command's output into the `terraform apply` log. It uses Terraform core
  # only (no extra providers), so it also works in the dev_overrides trial.
  provisioner "remote-exec" {
    inline = [
      "echo \"[$(hostname)] kernel $(uname -r) | $(nproc) vCPU | $(free -m | awk '/Mem:/{print $2}') MB RAM\"",
    ]
    connection {
      type        = "ssh"
      host        = self.primary_ip
      user        = "root"
      private_key = file("~/.ssh/id_ed25519")
      timeout     = "3m"
    }
  }
}

# name => primary IP for the whole fleet.
output "fleet" {
  value = { for vm in palmahost_compute_vm.node : vm.name => vm.primary_ip }
}

# `terraform destroy` terminates all fleet_size VMs in one step.
