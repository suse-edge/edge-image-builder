# Disable default mounts during RPM resolution
This documentation covers the design implementation for the logic introduced as part of [#235](https://github.com/suse-edge/edge-image-builder/pull/235).

## Problem
By default, when installing on a SLE OS, `podman` comes with a preconfigured `mounts.conf` file in both `/usr/share/containers/mounts.conf` and `/etc/containers/mounts.conf`. This `mounts.conf` file instructs `podman` to auto-mount any `/SRC:/DEST` that are present in the file to any containers created in the OS (more info in the [containers-mounts.conf.5.md](https://github.com/containers/common/blob/v0.57/docs/containers-mounts.conf.5.md) documentation). In the SLE OS case, the contents of the `mounts.conf` file point to the `/etc/SUSEConnect` and `/etc/zypp/credentials.d/SCCcredentials` files, which results in having the registration information mounted on any container that is created within the system. 

This is not desired during the RPM resolution process. Mainly because the RPM resolver image will be doing the registation with the specified `sccRegistationCode` for the specified OS. Meaning that if the `/etc/SUSEConnect` and `/etc/zypp/credentials.d/SCCcredentials` files have been auto-mounted with a registation data for a different OS, the registation that the RPM resolver tries to do will fail.

An example of a problem where the Resolver container cannot connect to the correct registation server, because an RMT server has been configured in the auto-mounted files can be seen below:

```bash
Registering system to registration proxy https://rmt.devlab.pgu1.suse.com/

Announcing system to https://rmt.devlab.pgu1.suse.com/ ...
SUSEConnect error: Post "https://rmt.devlab.pgu1.suse.com/connect/subscriptions/systems": x509: certificate signed by unknown authority
```  

The workflow of why this error occured can be seen below:
1. User runs a registered SLE OS as a host
1. `podman` is installed on this host, meaning that a `mounts.conf` file is present on both the default (`/usr/share/containers/mounts.conf`) and override (`/etc/containers/mounts.conf`) container file paths of the system. The contents of the `mounts.conf` contain the registration information of the host OS.
1. User builds the EIB image (using `bci-base` as the base image) or runs the provided EIB image. The build/run should be done as `root`, otherwise there will not be enough permissions to auto-mount the `mounts.conf` files and this use-case will not be hit
1. During the EIB image build/run, the `/etc/SUSEConnect` and `/etc/zypp/credentials.d/SCCcredentials` files are auto-mounted to the EIB container
1. Because the EIB image runs on top of `bci-base` and has `podman` installed, it also has the `mounts.conf` configuration, resulting in the `/etc/SUSEConnect` and `/etc/zypp/credentials.d/SCCcredentials` files being auto-mounted on top of the Resolver image that is being built inside the EIB container
1. Because the Resolver image attempts to register a different OS and does not have the needed permissions to the RTM server configured in the mounted `/etc/SUSEConnect` and `/etc/zypp/credentials.d/SCCcredentials` files, the RPM resolution process fails with the aforementioned error

**_Note: The example above is done to illustrate the problem. This can also happen with a regular SCC registation._**

## Solution
The least invasive solution is to programatically create an empty `mounts.conf` file at the expected override file path, which is `/etc/containers/mounts.conf`. When a `mounts.conf` file is present here it automatically overrides any other existing auto-mounts configured at the default mount place (`/usr/share/containers/mounts.conf`). 

The solution workflow is as follows:
1. Do a check whether there is an existing `mounts.conf` under `/etc/containers/mounts.conf`
2. If there is, rename it to `mounts.conf.orig`
3. Create an empty `mounts.conf` file

This solution also offers a way to re-enable the default auto-mounts by either returning the original `mounts.conf` file under `/etc/containers/mounts.conf`, or if this file does not exists, removing the empty blocking `mounts.conf` file. 

Since the `mounts.conf` file is not mounted, but present on each OS, we can safely do manipulations over this file without having to worry that we are breaking different systems. Furthermore, by not explicitly removing the original files we can ensure that even at code failure we will be able to revert to the original file configurations ensuring the least invasive manipulations possible.

## Other approaches
This section covers different approaches that were tried during the troubleshooting of the issue. It also explains the drawbacks of each approach.

### Use `suseconnect --url` property to provide correct registation server
Specifying the `suseconnect --url <registration_server>` is enough to change the registration server, but because the `/etc/SUSEConnect` and `/etc/zypp/credentials.d/SCCcredentials` files are still there, `suseconnect` treats the system as registered and we get the following error:
```bash
Error: Invalid system credentials, probably because the registered system was deleted in SUSE Customer Center. Check https://scc.suse.com whether your system appears there. If it does not, please call SUSEConnect --cleanup and re-register this system.
```    

### Clean up the system using `suseconnect --cleanup`
Because the `/etc/SUSEConnect` and `/etc/zypp/credentials.d/SCCcredentials` files are mounted, they cannot be removed and the following error can be seen:
```bash
SUSEConnect error: remove /etc/zypp/credentials.d/SCCcredentials: device or resource busy
```  

### Unmount the `/etc/SUSEConnect` and `/etc/zypp/credentials.d/SCCcredentials` files
Files seem to be mounted during the `podman build ...` command of the RPM resolver image. So unmounting them from within the `Dockerfile` will always result in an `umount: /etc/zypp/credentials.d/SCCcredentials: must be superuser to unmount.` error. The only way to do the unmount would be to unmount the files from within the EIB container before starting the Resolver image build. This is a very disruptive process, as we cannot revert the state once we are finished with the Resolver image build.

### Change the default mounts configuration in the [`containers.conf`](https://github.com/containers/common/blob/v0.57/docs/containers.conf.5.md) file
Doing this is possible, however it applies to the Podman VM machine. So a Podman machine would have to be created, but because we are running a rootful container we get the `Error: cannot run command "podman machine init" as root` error. Theoretically this might be achievable, but again this is a disruptive process, because we would have to write to a config file that contains much more configurations than just the auto-mount onеs and we could accidentally brake something else..

### Use a `podman` CLI flag to disable/change the default auto-mounts
Podman offers the **hidden** [`--default-mounts-file`](https://github.com/containers/podman/blob/v4.8.3/cmd/podman/root.go#L537) global flag (hidden [here](https://github.com/containers/podman/blob/v4.8.3/cmd/podman/root.go#L598)), that is not documented anywhere inside of Podman's documentation. When you pass the aforementioned flag to Podman's build command (`podman --default-mounts-file build ..`) the auto-mounts are overridden and the build process passes successfully. However, this flag is hidden, because it is intended for development/testing purposes only. Without going into the full code implementation, this flag translates to the [`MountsWithUIDGID`](https://github.com/containers/common/blob/v0.57/pkg/subscriptions/subscriptions.go#L168) function (when running it through the CLI), which clearly states that the change of the default auto-mount points is for [testing](https://github.com/containers/common/blob/v0.57/pkg/subscriptions/subscriptions.go#L175) purposes only. Furthermore, even if this was possible, it is only possible through the CLI. The same property is not enabled (parsed) from within the public podman library. The public podman image build [function](https://github.com/containers/podman/blob/v4.8.3/pkg/bindings/images/build.go#L53) accepts  [types.BuildOptions](https://github.com/containers/podman/blob/v4.8.3/pkg/domain/entities/types.go#L112) which provides the  [buildah.BuildOptions](https://github.com/containers/buildah/blob/v1.33.2/define/build.go#L115) configuration which offers a property that overrides the auto-mount directory([here](https://github.com/containers/buildah/blob/v1.33.2/define/build.go#L246)) however this property is not added as a `parameter` in Podman's build function, resulting in it not being used. This has been done, as the property is intended for development/testing purposes only.

### Search for a similar property to `--default-mounts-file` in Podman
Since for the RPM resolution this needs to happen during the image build, no other property (apart from the `--default-mounts-file`) that can do the desired change was found.
