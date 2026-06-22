output "resource_group_name" {
  description = "Name of the created resource group."
  value       = azurerm_resource_group.main.name
}

output "public_ip_address" {
  description = "Public IP of the VM -- use this for SSH and for reaching Grafana/Prometheus/the app."
  value       = azurerm_public_ip.main.ip_address
}

output "vm_name" {
  description = "Name of the VM."
  value       = azurerm_linux_virtual_machine.main.name
}

output "ssh_connection_command" {
  description = "Ready-to-use SSH command."
  value       = "ssh ${var.admin_username}@${azurerm_public_ip.main.ip_address}"
}