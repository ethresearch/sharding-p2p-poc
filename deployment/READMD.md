# Deployment

To understand how p2p nodes perform in real environments, we need to:

1. Deploy them on machines
2. Execute command for each machine.


## Requirements

1. Terraform
2. Ansible
3. terraform-inventory `brew install terraform-inventory`


## Getting started

1. Use Terraform to spin up some AWS instances for us.

```bash
cd terraform
terraform apply
```

2. Run commands with Ansible-playbook

```bash
cd ansible
export TF_STATE=../terraform/terraform.tfstate
ansible-playbook --inventory-file=$(which terraform-inventory) example.yml
```