variable "public_key_path" {
  default = "~/.ssh/id_rsa.pub"
}

variable "key_name" {
  default = "sharding_sim"
}

variable "tags" {
  type = "map"

  default = {
    Name = "Sharding simulation"
  }
}
