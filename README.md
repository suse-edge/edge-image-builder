# Edge Image Builder (EIB)

## Building

EIB is intended to run inside a container. Some form of container build tool and runtime are needed,
such as [Podman](https://podman.io/).

Build the container (from the root of this project):
```shell
podman build -t eib:dev .
```

## Running

At runtime, a volume must be mounted to the container. This serves as both the mechanism to introduce image
configuration files and provide a way to get the built image out of the container and on to the host machine.

**NOTE:** These docs are incomplete and will be fleshed out as the project matures. Below is an example that
is being used for dev purposes. At some point when it's more mature, an example configuration directory will be
added to this repository.

Example image configuration directory:
```shell
.
├── eib-config.yaml
└── images
    └── SLE-Micro.x86_64-5.5.0-Default-SelfInstall-GM.install.iso
```

That directory must be attached to the container at runtime. The following command attaches the directory and runs
EIB against the volume (replace `$IMAGE_DIR` with your local configuration directory):
```shell
podman run --rm -it -v $IMAGE_DIR:/eib eib:dev /bin/eib -config-file /eib/eib-config.yaml -config-dir /eib -build-dir /eib/_build
```

The command above will write all build artifacts (such as the combustion directory) under the image configuration
directory at `_build`. This likely won't be the default behavior and was included in the above example to ease
debugging.
