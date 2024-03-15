# Edge Image Builder Downloads

## EIB Builder Image Container Downloads

These packages are necessary for building the EIB binary.
```
gpgme-devel
device-mapper-devel
libbtrfs-devel
```

## EIB Deliverable Image Container Downloads

These packages are used by EIB to build the RTD image. Some of these package binaries may be copied to the RTD image.

Repository
```
https://download.opensuse.org/repositories/isv:SUSE:Edge:EdgeImageBuilder/SLE-15-SP5/isv:SUSE:Edge:EdgeImageBuilder.repo
```
Packages
```
xorriso
squashfs
libguestfs
kernel-default
e2fsprogs
parted
gptfdisk
btrfsprogs
guestfs-tools
lvm2
podman
createrepo_c
helm
hauler
nm-configurator 
```

## EIB Deliverable Image Programmatic Downloads

These artifacts are programmatically downloaded by EIB during build time. Some of these artifacts may be copied to the RTD image.

### Elemental
Source
```
https://download.opensuse.org/repositories/isv:/Rancher:/Elemental:/Maintenance:/5.5/standard/
```
Packages
```
elemental-register
elemental-system-agent
```

### RKE2/K3s
#### Releases 

Sources
```
https://github.com/rancher/rke2/releases/download/
https://github.com/k3s-io/k3s/releases/download/
```
Artifacts
```
RKE2 Binary
RKE2 Core Images
RKE2 Checksums
RKE2 Calico Images
RKE2 Canal Images
RKE2 Cilium Images
RKE2 Multus Images

K3s Binary
K3s Images
```

#### Installation Script

Sources
```
https://get.rke2.io
https://get.k3s.io
```
Artifacts
```
RKE2 Install Script
K3s Install Script
```

#### SELinux

Sources
```
https://rpm.rancher.io/k3s/stable/common/slemicro/noarch
https://rpm.rancher.io/rke2/stable/common/slemicro/noarch
```
Artifacts
```
rke2-selinux
k3s-selinux
```