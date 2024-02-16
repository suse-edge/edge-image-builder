# Edge Image Builder (EIB)

## Building

EIB is intended to run inside a container. Some form of container build tool and runtime are needed,
such as [Podman](https://podman.io/).

### Prerequisites
Before building the EIB image, make sure that you have the development headers and libraries for **gpgme**, **device-mapper** and **libbtrfs** installed on your system:

**SUSE Linux:**
```shell
sudo zypper install -y gpgme-devel device-mapper-devel libbtrfs-devel
```

**Ubuntu:** 
```shell
sudo apt-get install -y libgpgme-dev libdevmapper-dev libbtrfs-dev
```

**Fedora:**
```shell
sudo dnf -y install gpgme-devel device-mapper-devel btrfs-progs-devel
```

Build the container (from the root of this project):
```shell
podman build -t eib:dev .
```

## Running

**NOTE:** These docs are incomplete and will be fleshed out as the project matures. At some point when it's
more mature, an example configuration directory will be added to this repository.

### Image Definition

For details on how to create the configuration artifacts needed to build an image, see the
[Building Images](docs/building-images.md) guide.

### Running EIB

The image configuration directory must be attached to the container at runtime. This serves as both the mechanism
to introduce image configuration files and provide a way to get the built image out of the container and onto
the host machine. 

The following example command attaches the directory and runs EIB:
```shell
podman run --rm -it \
-v $IMAGE_DIR:/eib eib:dev /bin/eib build \
--config-file $CONFIG_FILE.yaml \
--config-dir /eib \
--build-dir /eib/_build
```

**NOTE:**
Elemental builds require the ```--privileged``` flag. Please see here why this is required [Package resolution design](./docs/design/pkg-resolution.md#running-the-eib-container).
```shell
podman run --rm --privileged -it \
-v $IMAGE_DIR:/eib eib:dev /bin/eib build \
--config-file $CONFIG_FILE.yaml \
--config-dir /eib \
--build-dir /eib/_build
```

* `-v` - Used to mount a local directory (in this example, the value of $IMAGE_DIR) into the EIB container at `/eib`.
* `--config-file` - Specifies which image configuration file to build. The path to this file will be relative to
  the image configuration directory. If the configuration file is in the root of the configuration directory, simply 
  specify the name of the configuration file 
* `--config-dir` - Specifies the image configuration directory. Keep in mind that this is relative to the running
  container, so its value must match the mounted volume.
* `--build-dir` - (optional) If unspecified, EIB will use a temporary directory inside the container for
  assembling/generating the components used in the build. This may be specified to a location within the mounted
  volume to make the build artifacts available after the container completes. In this example, a directory named
  `_build` will be created in the image configuration directory and will persist after EIB finishes. This directory
  will contain subdirectories storing the respective artifacts of the different builds.
* `--validate` - If specified, the specified image definition and configuration directory will be checked to ensure
  the build can proceed, however the image will not actually be built.

> **_NOTE:_** When specifying packages for installation, you need to also pass the [`--privileged`](https://docs.podman.io/en/latest/markdown/podman-run.1.html#privileged) option as well. For more information on why this is needed, see the [Package resolution design](./docs/design/pkg-resolution.md#running-the-eib-container) documentation.

## Testing Images

For details on how to test the built images, see the [Testing Guide](docs/testing-guide.md).
