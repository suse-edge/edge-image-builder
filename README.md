# Edge Image Builder (EIB)

## Usage

EIB runs as a container. Some form of container runtime is needed, such as [Podman](https://podman.io/).

The latest version of EIB (1.2.0) can be downloaded from the Open Build Service using the following command:

```bash
podman pull registry.suse.com/edge/3.3/edge-image-builder:1.2.0
```

Alternatively, EIB can be built from this repository. See the [Building from Source](#building-from-source)
section below.

### Image Definition

For details on how to create the artifacts needed to build an image, see the
[Building Images](docs/building-images.md) guide.

### Running EIB

The image configuration directory must be attached to the container at runtime. This serves as both the mechanism
to introduce image definition files and provide a way to get the built image out of the container and onto
the host machine.

#### Validating an image definition

The following example command attaches the image configuration directory and validates a definition:
```shell
podman run --rm -it -v $IMAGE_DIR:/eib \
$EIB_IMAGE \
validate --definition-file $DEFINITION_FILE.yaml
```

* `-v` - Used to mount a local directory (in this example, the value of $IMAGE_DIR) into the EIB container at `/eib`.
* `--definition-file` - Specifies which image definition file to build. The path to this file will be relative to
  the image configuration directory. If the definition file is in the root of the configuration directory, simply
  specify the name of the configuration file.
* `--config-dir` - (Optional) Specifies the image configuration directory. This path is relative to the running container, so its
  value must match the mounted volume. It defaults to `/eib` which matches the mounted volume `$IMAGE_DIR:/eib` in the example above.

#### Building an image

The following example command attaches the image configuration directory and builds an image:
```shell
podman run --rm -it -v $IMAGE_DIR:/eib \
$EIB_IMAGE \
build --definition-file $DEFINITION_FILE.yaml
```

**NOTE:**
Image builds which involve package resolution must include the [`--privileged`](https://docs.podman.io/en/latest/markdown/podman-run.1.html#privileged)
flag. Package resolution will be automatically performed when requesting package installation or configuring components
which require it (e.g. Elemental, Kubernetes SELinux, etc.).

* `-v` - Used to mount a local directory (in this example, the value of $IMAGE_DIR) into the EIB container at `/eib`.
* `--definition-file` - Specifies which image definition file to build. The path to this file will be relative to
  the image configuration directory. If the definition file is in the root of the configuration directory, simply 
  specify the name of the configuration file.
* `--config-dir` - (Optional) Specifies the image configuration directory. This path is relative to the running container, so its
  value must match the mounted volume. It defaults to `/eib` which matches the mounted volume `$IMAGE_DIR:/eib` in the example above.
* `--build-dir` - (Optional) If unspecified, EIB will create a `_build` directory under the image configuration directory 
  for assembling/generating the components used in the build which will persist after EIB finishes. This may also be
  specified to another location within a mounted volume. The directory will contain subdirectories storing the
  respective artifacts of the different builds as well as cached copies of certain downloaded files.

## Testing Images

For details on how to test the built images, see the [Testing Guide](docs/testing-guide.md).

## Building from Source

Build the container (from the root of this project). The image tag `eib:dev`
will be used in the Podman command examples above for the `$EIB_IMAGE` variable.

```shell
podman build -t eib:dev .
```