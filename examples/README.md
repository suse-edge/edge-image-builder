# Image Definition Examples

This directory provides a number of example image definition files that can be used
directly with Edge Image Builder. They are divided across two directories depending on the
resulting image type (ISO or RAW image).

Unless otherwise specified, these definitions may be used as-is with one caveat. The base
image in your image configuration directory must match the name specified in the `baseImage` field
of the definition. This can be achieved either through renaming your local image or changing this
field in the definition itself.

> :warning: For simplicity in testing, many of these definitions include defaults that are not suitable
> for production (i.e. setting a simple password for `root`). Please exercise caution in copying snippets
> of these examples for production uses.

---

## Setup

The definitions included in this directory are slightly opinionated for organizational purposes. These
opinions include:

* A directory under the image configuration directory named `definitions` in which all of these definitions 
  will live. This is not strictly needed for EIB, but if you are working with multiple definitions, it can
  greatly simplify the configuration directory.
* A directory under the image configuration directory named `out` in which all built images will be stored.
  EIB will put built images relative to the image configuration directory, so simply prefixing the output
  image name with `out/` allows us to organize all of the built images into a subdirectory.

Following these conventions, the following is the bare minimum image configuration directory needed to
run the example definitions:

```bash
.
├── base-images
├── definitions
│   ├── iso
│   └── raw
└── out
```

The directory structure above can be created using the following command:

```bash
mkdir -p {base-images,definitions/iso,definitions/raw,out}
```

The following shows an example of the above directory structure, populated with the base images, definitions,
and the results of performing multiple builds:

```bash
.
├── base-images
│   ├── SLE-Micro.x86_64-5.5.0-Default-GM.raw
│   ├── SLE-Micro.x86_64-5.5.0-Default-SelfInstall-GM2.install.iso
├── definitions
│   ├── iso
│   │   └── basic.yaml
│   └── raw
│       └── basic.yaml
└── out
    ├── basic.raw
    └── basic.iso
```

With this structure, the EIB run command should be run from the root of that directory, using the relative path
to the desired definition. For example:

```bash
podman run --rm --privileged -it \
  -v .:/eib eib:dev build \
  --definition-file ./definitions/iso/basic.yaml
```

---

## Simple Examples

### `iso/basic.yaml`

| Option       | Default Value                                                           |
|--------------|-------------------------------------------------------------------------|
| Base Image   | `base-images/SLE-Micro.x86_64-5.5.0-Default-SelfInstall-GM.install.iso` |
| Output Image | `out/basic.iso`                                                         |

* Configures the `root` password to be `slemicro`.
* Configures the ISO installation to run unattended, meaning there will be no required user input for
  the installer questions (i.e. selecting the "Install" option, opting to delete the installation device).
  * This requires a patched version of SLE Micro that is not yet publicly available; without this version
    the build will complete successfully but the user will be prompted for input before the installation
    continues.
  * This definition defaults the installation device to `/dev/vda` which works with libvirt. Depending on your
    setup, this may need to be tweaked.

---

### `raw/basic.yaml`

| Option       | Default Value                                       |
|--------------|-----------------------------------------------------|
| Base Image   | `base-images/SLE-Micro.x86_64-5.5.0-Default-GM.raw` |
| Output Image | `out/basic.raw`                                     |

* Configures the `root` password to be `slemicro`.

---

## Advanced Examples

Examples in this section require more setup than a simple image definition file and base image. This
section will describe the necessary image configuration directory structure and supplemental files 
in order to run each definition.
