# Deployment

To understand how p2p nodes perform in real environments, we need to:

1. Deploy them on machines
2. Execute command for each machine.


## Requirements

1. Terraform
2. Ansible


## Getting started

1. Use Terraform to spin up some AWS instances for us.

```bash
cd terraform
terraform apply
sh ansible_inventory_from_terraform_state.sh
```
This dumps the output of terraform state as an ansible inventory file.

2. Run commands with Ansible-playbook

```bash
cd ansible
ansible-playbook example.yml
```