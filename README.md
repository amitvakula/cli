# Flywheel CLI

# Building

```
git clone git@github.com:flywheel-io/cli.git; cd cli

./make.sh
```

The binary will be compiled to `.gopath/bin/fw`.

# Interacting with a Flywheel instance

First, you need to generate an API key via your profile page.
Login using the CLI with the URL of the site and your API key:

```
$ fw login https://localhost:8443 TeFwKZ01-qLGtF2F9UmOIt4BuMjLysGMvbmQ3z2Z-BFoeoADs4imldee
Logged in as Nathaniel Kofalt <nathanielkofalt@flywheel.io>
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
