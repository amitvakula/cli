package gears

const (
	ApiFailedMsg = "Could not access the Flywheel instance.\nCheck that you're online and that your API key is up to date (try 'fw login')."

	CreateContainerFailedMsg = "Could not access the specified docker image.\nCheck that you're online and that the docker image is correct (try 'docker pull' or similar)."

	CopyFromContainerFailedMsg = "Could set up the specified docker image.\nCheck that you're online and that the docker image is correct (try 'docker pull' or similar)."

	UntarFailedMsg = "Could not parse the manifest file.\nCheck that the manifest file in the image is correct - for example, an extra or missing comma."

	JsonFailedMsg = "Could prepare a new manifest file.\nCheck that the manifest file in the image is correct - for example, an extra or missing comma."

	WriteFailedMsg = "Could not write file to current folder.\nCheck permissions and that the disk is not full."
)
