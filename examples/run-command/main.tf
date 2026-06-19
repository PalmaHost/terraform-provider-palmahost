terraform {
  required_providers {
    palmahost = { source = "palmahost/palmahost" }
    external  = { source = "hashicorp/external" } # for capturing command output
  }
}
provider "palmahost" {} # PALMAHOST_TOKEN env; base_url defaults to my.palmahost.sh

variable "fleet_size" {
  type    = number
  default = 4
}

variable "command" {
  type        = string
  default     = "uptime -p"
  description = "Command to run on every node."
}

resource "palmahost_ssh_key" "deploy" {
  name       = "run-command-demo"
  public_key = file("~/.ssh/id_ed25519.pub")
}

resource "palmahost_compute_vm" "node" {
  count    = var.fleet_size
  name     = "node-${count.index + 1}"
  image    = "ubuntu-24.04"
  plan     = "pc2-1c-1g"
  location = "es"
  ssh_key  = palmahost_ssh_key.deploy.id
}

# SSH into each node, run `command`, and return its stdout as a value. Unlike a
# remote-exec provisioner (which only logs to the apply output), an external
# data source captures the result so you can output or feed it into other
# resources. Needs `terraform init` (the hashicorp/external provider) plus
# `jq` and `ssh` on the machine running Terraform.
data "external" "run" {
  count = var.fleet_size
  program = ["bash", "-c", <<-EOT
    out=$(ssh -o StrictHostKeyChecking=no -o ConnectTimeout=15 \
      -i ~/.ssh/id_ed25519 \
      root@${palmahost_compute_vm.node[count.index].primary_ip} \
      ${jsonencode(var.command)} 2>&1)
    jq -n --arg result "$out" '{"result": $result}'
  EOT
  ]
}

# node name => command output.
output "results" {
  value = {
    for i, vm in palmahost_compute_vm.node :
    vm.name => trimspace(data.external.run[i].result["result"])
  }
}
