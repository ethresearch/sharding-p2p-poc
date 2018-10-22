data "template_file" "nodes_ansible" {
  count    = "${var.cluster_size}"
  template = "${file("${path.module}/template/hostname.tpl")}"

  vars {
    extra = "${var.vm_user}@${element(aws_instance.nodes_hosts.*.public_ip,count.index)} p2p_nodes_start=${30399 + count.index*4} p2p_nodes_end=${30402 + count.index*4}"
  }
}

data "template_file" "log_collector_ansible" {
  template = "${file("${path.module}/template/hostname.tpl")}"

  vars{
    extra = "${var.vm_user}@${aws_instance.log_collector.public_ip}"
  }
}

data "template_file" "ansible_inventory" {
  template = "${file("${path.module}/template/ansible_inventory.tpl")}"

  vars {
    nodes_hosts = "${join("\n",data.template_file.nodes_ansible.*.rendered)}"
    log_collector = "${data.template_file.log_collector_ansible.rendered}"
  }
}

output "ansible_inventory" {
  value = "${data.template_file.ansible_inventory.rendered}"
}
