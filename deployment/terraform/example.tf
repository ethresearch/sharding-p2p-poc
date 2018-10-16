provider "aws" {
  region = "us-east-1"
}

resource "aws_key_pair" "auth" {
  key_name   = "${var.key_name}"
  public_key = "${file(var.public_key_path)}"
}

resource "aws_instance" "example" {
  
  ami           = "ami-059eeca93cf09eebd"
  instance_type = "t2.micro"
  tags          = "${var.tags}"
  key_name      = "${var.key_name}"
  count         = "${var.cluster_size}"

  provisioner "remote-exec" {
    # The connection will use the local SSH agent for authentication
    inline = ["echo Successfully connected"]

    # The connection block tells our provisioner how to communicate with the resource (instance)
    connection {
      user = "${var.vm_user}"
    }
  }
}
