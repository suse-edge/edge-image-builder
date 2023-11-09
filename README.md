# Edge Image Builder (EIB)

## Building

EIB is intended to run inside a container. Some form of container build tool and runtime are needed,
such as [Podman](https://podman.io/).

Build the container (from the root of this project):
```shell
podman build -t eib:dev .
```

## Running

**NOTE:** These docs are incomplete and will be fleshed out as the project matures. Below is an example that
is being used for dev purposes. At some point when it's more mature, an example configuration directory will be
added to this repository.

### Image Definition

Two things are necessary to build an image using EIB:
1. A configuration file that describes the image to build
1. A directory that contains the base SLE Micro image to modify, along with any other custom files that
   will be included in the built image

#### Image Configuration File

The image configuration file is a YAML document describing a single image to build. The file is specified using
the `-config-file` argument. Only a single image may be built at a time, however the same image configuration
directory may be used to build multiple images by creating multiple configuration files.

The following can be used as the minimum configuration required to create an image:
```yaml
apiVersion: 1.0
image:
  imageType: iso
  baseImage: SLE-Micro.x86_64-5.5.0-Default-SelfInstall-GM.install.iso
  outputImageName: eib-image.iso
```

* `imageType` - Must be either `iso` or `raw`.
* `baseImage` - Indicates the name of the image file used as the base for the built image. This file must be located
  under the `images` directory of the image configuration directory (see below for more information). This image will
  **not** directly be modified by EIB; a new image will be created each time EIB is run.
* `outputImageName` - Indicates the name of the image that EIB will build. This may only be a filename; the image will
  be written to the root of the image configuration directory.

#### Image Configuration Directory

The image configuration directory contains all the files necessary for EIB to build an image. As the project matures,
the structure of this directory will be better fleshed out. For now, the required structure is described below:

```shell
.
├── eib-config-iso.yaml
├── eib-config-raw.yaml
└── images
    └── SLE-Micro.x86_64-5.5.0-Default-SelfInstall-GM.install.iso
    └── SLE-Micro.x86_64-5.5.0-Default-GM.raw
```

* `eib-config-iso.yaml`, `eib-config-raw.yaml` - All image configuration files should be in the root of the image 
  configuration directory. Multiple configuration files may be included in a single configuration directory, with 
  the specific configuration file specified as a CLI argument as described above.
* `images` - This directory must exist and contains the base images from which EIB will build customized images. There
  are no restrictions on the naming; the image configuration file will specify which image in this directory to use
  for a particular build.

There are a number of optional directories that may be included in the image configuration directory:

* `scripts` - If present, all the files in this directory will be included in the built image and automatically
  executed during the combustion phase.

### Running EIB

The image configuration directory must be attached to the container at runtime. This serves as both the mechanism
to introduce image configuration files and provide a way to get the built image out of the container and onto
the host machine. 

The following example command attaches the directory and runs EIB:
```shell
podman run --rm -it \
-v $IMAGE_DIR:/eib eib:dev /bin/eib \
-config-file $CONFIG_FILE.yaml \
-config-dir /eib \
-build-dir /eib/_build
```

* `-v` - Used to mount a local directory (in this example, the value of $IMAGE_DIR) into the EIB container at `/eib`.
* `-config-file` - Specifies which image configuration file to build. The path to this file will be relative to
  the image configuration directory. If the configuration file is in the root of the configuration directory, simply 
  specify the name of the configuration file 
* `-config-dir` - Specifies the image configuration directory. Keep in mind that this is relative to the running
  container, so its value must match the mounted volume.
* `-build-dir` - (optional) If unspecified, EIB will use a temporary directory inside the container for
  assembling/generating the components used in the build. This may be specified to a location within the mounted
  volume to make the build artifacts available after the container completes. In this example, a directory named
  `_build` will be created in the image configuration directory and will persist after EIB finished. If it already
  exists, EIB will delete this directory at the start of a build, meaning it does not need to manually be cleaned up
  between builds.

## Testing Images

For details on how to test the built images, see the [Testing Guide](docs/testing-guide.md).