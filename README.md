# dicom-workshop
## Building

```bash
git clone https://github.com/flywheel-io/dicom-workshop workspace/src/flywheel.io/dicom
ln -s workspace/src/flywheel.io/dicom dicom-workspace

./dicom-workspace/make.sh
```

This builds the golang dicom uploader.<br/>

## Use

```bash
dicom -folder [folder path] -group [group id] -project [project label]
```
