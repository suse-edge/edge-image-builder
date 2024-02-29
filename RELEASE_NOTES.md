# Edge Image Builder Releases

# v1.0.0-rc1

* Added support for deploying user-provided Helm charts
* Added support for custom network configuration scripts

## Image Definition Changes

* Removed the `embeddedArtifactRegistry/images/supplyChainKey` attribute
* Changed `operatingSystem/users/sshKey` into `operatingSystem/users/sshKeys` and it is now a list instead of a single string
* Added the ability to configure operating system groups under `operatingSystem/groups`
* Added optional `primaryGroup` field for operating system users
* Added optional `secondaryGroups` field for operating system users
* Added optional `createHomeDir` field for operating system users
* Added optional `uid` field for operating system users

## Bug Fixes

* #197 - Consider using ENTRYPOINT instead of CMD
* #213 - zypper clean after zypper install
* #216 - Update the docs to reflect that systemd can be used for any kind of systemd unit, not just services
