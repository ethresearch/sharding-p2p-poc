provider "aws" {
  region = "us-east-1"
}

locals {
  # The default username for our AMI
  vm_user = "ubuntu"
}

resource "aws_key_pair" "auth" {
  key_name   = "${var.key_name}"
  public_key = "${file(var.public_key_path)}"
}

resource "aws_instance" "example" {
  
  ami           = "ami-06b5810be11add0e2"
  instance_type = "t2.micro"
  tags          = "${var.tags}"
  key_name      = "${var.key_name}"
  count         = 3

  provisioner "remote-exec" {
    # The connection will use the local SSH agent for authentication
    inline = ["echo Successfully connected"]

    # The connection block tells our provisioner how to communicate with the resource (instance)
    connection {
      user = "${local.vm_user}"
    }
  }
}
