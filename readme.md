
# Flywheel Command-Line Tool

[![Build status](https://circleci.com/gh/flywheel-io/cli/tree/master.svg?style=shield&circle-token=fa0c0bf6fa27a8548231fc12baff5f633ae201d8)](https://circleci.com/gh/flywheel-io/cli)

## Building

### Cloning

This project expects to be in your GOPATH. E.g. using workspace:
```
git clone git@github.com:flywheel-io/cli workspace/src/flywheel.io/fw
ln -s workspace/src/flywheel.io/fw cli
```

### Building Python

The CLI executable wraps a python virtual environment and standalone interpreter.
In order to build the executable, you'll need to build the python portion of it first.
This requires Python 3.6 (e.g. with a virtual environment activated)

```bash
./cli/make.sh buildPython
```

### Building Executable

```bash
./cli/make.sh
```

The binary will be compiled to `workspace/bin/fw`.

## Interacting with a Flywheel instance

First, you need to generate an API key via your profile page.
Login using the CLI with the URL of the site and your API key:

```
$ fw login dev.flywheel.io:Xz6SLBbDFu0Zne6uA1
Logged in as Nathaniel Kofalt!
```

These credentials will be stored in `~/.config/flywheel`.
You can now explore and download files from the storage hierarchy:

```
$ fw ls
scitran Scientific Transparency

$ fw ls scitran
Testdata
Neuroscience
Psychology

$ fw ls scitran/Neuroscience
patient_2
patient_1
control_1
control_2
patient_343

$ fw ls scitran/Neuroscience/patient_1
8403_6_1_fmri
8403_4_1_t1
8403_1_1_localizer

$ fw ls scitran/Neuroscience/patient_1/8403_1_1_localizer
8403_1_1_localizer.dicom.zip

$ fw download scitran/Neuroscience/patient_1/8403_1_1_localizer/8403_1_1_localizer.dicom.zip
```

## Choosing a Python CLI Version

The python portion of the CLI is grabbed via PIP. You can update update which
version to pull by updating `python-cli-version.txt`.

## Creating a release

When creating a new CLI release, update `python-cli-version.txt` and `fw.go` with the 
correct version before tagging.

