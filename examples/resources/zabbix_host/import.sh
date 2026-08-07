# Zabbix objects are imported by their numeric id, which the frontend shows
# in the URL of the object's configuration form.
terraform import zabbix_host.example 10634

# tls_psk_identity and tls_psk are write-only in the Zabbix API -- host.get
# never returns them -- so they import as empty. Re-add them to the
# configuration afterwards; the next apply sends them again.
