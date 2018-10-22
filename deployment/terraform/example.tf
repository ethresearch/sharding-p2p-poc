provider "aws" {
  region = "us-east-1"
}

resource "aws_key_pair" "auth" {
  key_name   = "${var.key_name}"
  public_key = "${file(var.public_key_path)}"
}

resource "aws_instance" "nodes_hosts" {

  ami           = "ami-059eeca93cf09eebd"
  instance_type = "t2.micro"
  tags          = "${var.tags}"
  key_name      = "${var.key_name}"
  count         = "${var.cluster_size}"
  security_groups=["${aws_security_group.sharding_sim.name}"]

  provisioner "remote-exec" {
    # The connection will use the local SSH agent for authentication
    inline = ["echo Successfully connected"]

    # The connection block tells our provisioner how to communicate with the resource (instance)
    connection {
      user = "${var.vm_user}"
    }
  }
}

resource "aws_instance" "log_collector" {

  ami           = "ami-059eeca93cf09eebd"
  instance_type = "t2.micro"
  tags          = "${var.collector_tags}"
  key_name      = "${var.key_name}"
  count         = 1
  security_groups=["${aws_security_group.sharding_sim.name}"]

  provisioner "remote-exec" {
    inline = ["echo Log collector successfully connected"]
    connection {
      user = "${var.vm_user}"
    }
  }
}

resource "aws_security_group" "sharding_sim" {
  name        = "Sharding simulation"
  tags = "${var.tags}"

  # SSH access from anywhere
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # HTTP access from anywhere
  ingress {
    from_port   = 0
    to_port     = 65535
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 0
    to_port     = 65535
    protocol    = "udp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # outbound internet access
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}