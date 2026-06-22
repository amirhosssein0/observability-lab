variable "project_name" {
  description = "Prefix used for naming every resource (e.g. obslab)."
  type        = string
  default     = "obslab"
}

variable "location" {
  description = "Azure region."
  type        = string
  default     = "Japan East"
}

variable "vm_size" {
  description = "VM SKU."
  type        = string
  default     = "Standard_D2s_v3"
}

variable "os_disk_size_gb" {
  description = "OS disk size in GB."
  type        = number
  default     = 64
}

variable "admin_username" {
  description = "Admin username for SSH login."
  type        = string
  default     = "azureuser"
}

variable "ssh_public_key_path" {
  description = "Path to your local SSH public key, e.g. ~/.ssh/id_rsa.pub."
  type        = string
}

variable "admin_source_ip" {
  description = "Your IP/CIDR allowed to reach the VM, e.g. \"1.2.3.4/32\". Run `curl ifconfig.me` to find yours. Avoid \"*\" once the lab is reachable -- fine temporarily, but tighten it."
  type        = string
  default     = "63.141.252.202/32"
}

variable "enable_auto_shutdown" {
  description = "Automatically stops the VM every day to control cost."
  type        = bool
  default     = true
}

variable "auto_shutdown_time" {
  description = "Daily shutdown time, 24h HHmm format (VM's region time)."
  type        = string
  default     = "2300"
}

variable "auto_shutdown_timezone" {
  description = "Timezone for the auto-shutdown schedule (Windows timezone ID)."
  type        = string
  default     = "UTC"
}

variable "tags" {
  description = "Tags applied to every resource."
  type        = map(string)
  default = {
    project     = "observability-lab"
    environment = "lab"
  }
}