# Configure GRUB for first boot
# - Without this, the values wouldn't be used until after the first time the
#   grub configuration is regenerated
download /boot/grub2/grub.cfg /tmp/grub.cfg
! sed -i '/ignition.platform/ s/$/ {{.KernelArgs}} /' /tmp/grub.cfg
upload /tmp/grub.cfg /boot/grub2/grub.cfg

# Configure GRUB defaults
# - Without this, when `transactional-update grub.cfg` is run it will overwrite
#   settings used in the above change
download /etc/default/grub /tmp/grub
! sed -i '/^GRUB_CMDLINE_LINUX_DEFAULT="/ s/"$/ {{.KernelArgs}} "/' /tmp/grub
upload /tmp/grub /etc/default/grub