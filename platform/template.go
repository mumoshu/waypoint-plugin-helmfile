package platform

import (
	"github.com/hashicorp/waypoint/builtin/docker"
)

// Most or all of the code in this file is borrowed from https://github.com/hashicorp/waypoint/blob/main/builtin/exec/platform.go

// tplData is the structure given to Go's text/template when rendering
// templates.
type tplData struct {
	// Input comes from the function input.
	Input map[string]interface{}

	// Env are environment variables that should be set. These MUST be
	// set for the entrypoint to work properly.
	Env map[string]string

	// Workspace is the workspace that this execution is running in.
	Workspace string
}

func (d *tplData) Populate(input *Input) {
	if input == nil {
		return
	}

	if d.Input == nil {
		d.Input = map[string]interface{}{}
	}

	for k, value := range input.Data {
		d.Input[k] = inputValueToInterface(value)
	}
}

func (d *tplData) PopulateImage(image *docker.Image) {
	if image == nil {
		return
	}

	d.Input["DockerImageName"] = image.GetImage()
	d.Input["DockerImageTag"] = image.GetTag()
}

func inputValueToInterface(v *Input_Value) interface{} {
	switch v := v.Value.(type) {
	case *Input_Value_Text:
		return v.Text

	default:
		return nil
	}
}
