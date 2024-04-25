# Edge Image Builder User Guide

## Building Images

The [Building Images Guide](./building-images.md) describes all of the possible customizations
Edge Image Builder (EIB) can apply to an image. This guide describes the necessary image definition
sections and image configuration directory structure for each configurable component and should serve
as a starting guide to new EIB users.

## Installing Packages in an Image

EIB provides the ability to define RPM repositories and indicate a list of packages to install on the built
image. The RPMs and their dependencies will be embedded in the resulting image and, on first boot, will
be installed on the running system. The [Installing Packages Guide](./installing-packages.md) contains
detailed information on how to configure an image definition to support this functionality.

## Testing Images

The [Testing Guide](./testing-guide.md) provides information on how to run the customized, ready to boot (CRB)
images produced by EIB in a virtual machine.

## Debugging

The [Debugging Guide](./debugging.md) describes the log files generated during a build and where to look
in the event of a failed build. This guide also contains information on the build directory generated when
EIB runs and the files it may contain.