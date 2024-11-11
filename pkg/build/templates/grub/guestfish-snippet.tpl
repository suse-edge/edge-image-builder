# Configure GRUB defaults
# - So that the update below, and later`transactional-update grub.cfg` will persist the changes
download /etc/default/grub /tmp/grub
# Remove original kernel arguments that match desired kernelArgs that have values
{{ range .KernelArgsList -}}
! sed -i "/^GRUB_CMDLINE_LINUX_DEFAULT/ s/$(echo {{ . }} | cut -f1 -d"=")=[^=]//" /tmp/grub
{{ end -}}
# Add in the new ones
! sed -i '/^GRUB_CMDLINE_LINUX_DEFAULT="/ s/"$/ {{.KernelArgs}} "/' /tmp/grub
upload /tmp/grub /etc/default/grub

# Configure GRUB for first boot
# - This re-generates the grub.cfg applying the /etc/default/grub above
sh "grub2-mkconfig -o /boot/grub2/grub.cfg"
