package gears

import (
	"errors"
	"os"
	"regexp"

	"github.com/manifoldco/promptui"

	. "flywheel.io/fw/util"
	"flywheel.io/sdk/api"
)

var (
	// https://github.com/flywheel-io/gears/blob/master/spec/manifest.schema.json

	gearLabelRegex     = regexp.MustCompile("^.{1,100}$")
	gearLabelRegexFail = errors.New("Gear label is invalid. Must be less than 100 characters.")
	gearLabelValidator = createValidator(gearLabelRegex, gearLabelRegexFail)
	gearLabelPrompt    = createPrompt("Name your gear", gearLabelValidator)

	gearNameRegex     = regexp.MustCompile("^[a-z0-9\\-]+$")
	gearNameRegexFail = errors.New("Gear name is invalid. Use only lower-case letters, numbers, and dashes (-).")
	gearNameValidator = createValidator(gearNameRegex, gearNameRegexFail)
	gearNamePrompt    = createPrompt("Name your gear", gearNameValidator)

	// gearImageList   = []string{"flywheel/example-gear", "flywheel/gear-base-anaconda", "flywheel/base-gear-ubuntu"}
	gearImageList   = []string{"python:3", "python:2", "ubuntu"}
	gearImagePrompt = createSelectWA("Select base image", gearImageList, "Other")

	gearCategoryList   = []api.GearCategory{api.ConverterGear, api.AnalysisGear}
	gearCategoryPrompt = createSelect("Select gear type", gearCategoryList)

	confirmReplaceManifestMsg = "File manifest.json exists and will be replaced. Continue?"
	confirmReplaceManifestPmt = createConfirm(confirmReplaceManifestMsg)

	confirmReplaceScriptMsg = "File example.sh exists and will be replaced. Continue?"
	confirmReplaceScriptPmt = createConfirm(confirmReplaceScriptMsg)

	confirmReplaceScriptMsgP = "File example.py exists and will be replaced. Continue?"
	confirmReplaceScriptPmtP = createConfirm(confirmReplaceScriptMsgP)
)

func createValidator(regex *regexp.Regexp, err error) promptui.ValidateFunc {
	return func(input string) error {
		if regex.MatchString(input) {
			return nil
		} else {
			return err
		}
	}
}

func createConfirm(label string) *promptui.Prompt {
	return &promptui.Prompt{
		Label:     label,
		IsConfirm: true,
	}
}

func createPrompt(label string, validator promptui.ValidateFunc) *promptui.Prompt {
	return &promptui.Prompt{
		Label:    label,
		Validate: validator,
	}
}

func createSelect(label string, items interface{}) *promptui.Select {
	return &promptui.Select{
		Label: label,
		Items: items,
	}
}
func createSelectWA(label string, items []string, addLabel string) *promptui.SelectWithAdd {
	return &promptui.SelectWithAdd{
		Label:    label,
		Items:    items,
		AddLabel: addLabel,
	}
}

func runConfirm(prompt *promptui.Prompt) bool {
	_, err := prompt.Run()

	// Confirm behavior is strange
	// https://github.com/manifoldco/promptui/issues/81
	if err != nil && err.Error() == "" {
		return false
	} else if err == nil {
		return true
	}

	Check(err)
	return false
}

func runConfirmFatal(prompt *promptui.Prompt) {
	if !runConfirm(prompt) {
		Println("Canceled.")
		Println()
		os.Exit(1)
	}
}

func runPrompt(prompt *promptui.Prompt) string {
	result, err := prompt.Run()
	Check(err)
	return result
}

func runSelect(prompt *promptui.Select) string {
	_, result, err := prompt.Run()
	Check(err)
	return result
}

func runSelectWA(prompt *promptui.SelectWithAdd) string {
	_, result, err := prompt.Run()
	Check(err)
	return result
}

// Returns gear label, name, image, and type
func gearCreatePrompts() (string, string, string, string) {
	Println()

	Println("Welcome! First, give your gear a friendly, human-readable name.")
	gearLabel := runPrompt(gearLabelPrompt)
	Println()

	Println("Next, specify a gear ID. This is a unique, machine-friendly abbreviation.")
	gearName := runPrompt(gearNamePrompt)
	Println()

	Println("Choose a Docker image to start your project.")
	baseImage := runSelectWA(gearImagePrompt)
	Println()

	Println("Is this a converter or an analysis gear?")
	gearType := runSelect(gearCategoryPrompt)
	Println()

	return gearLabel, gearName, baseImage, gearType
}
