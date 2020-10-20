package platform

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/waypoint-plugin-sdk/component"
	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
	"github.com/hashicorp/waypoint/builtin/docker"

	//"github.com/hashicorp/waypoint/builtin/docker"
	"github.com/hashicorp/waypoint/builtin/k8s"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

const (
	// https://github.com/hashicorp/waypoint/blob/main/builtin/k8s/platform.go
	labelId    = "waypoint.hashicorp.com/id"
	labelNonce = "waypoint.hashicorp.com/nonce"
)

// PlatformConfig is the configuration structure for the component.Platform,
// that contains the Helmfile config read from waypoint.hcl
type PlatformConfig struct {
	HelmfileVersion string `hcl:"helmfile_version,optional"`
	HelmVersion     string `hcl:"helm_version,optional"`
	HelmDiffVersion string `hcl:"helm_diff_version,optional"`
	HelmfileBin     string `hcl:"helmfile_bin,optional"`
	HelmBin         string `hcl:"helm_bin,optional"`
	Namespace       string `hcl:"namespace,optional"`

	// Path is the path to helmfile.yaml to be executed by Helmfile
	Path string `hcl:"path,optional"`

	// Dir is the working directory to set when executing the command.
	// This will default to the path to the application in the Waypoint
	// configuration.
	Dir string `hcl:"dir,optional"`

	Selectors []string `hcl:"selectors,optional"`

	// EnvironmentTemplate is the template to render for generating Helmfile environment name
	EnvironmentTemplate string `hcl:"environment_template,optional"`

	// ValuesTemplate is the template to render for generating Helmfile values from Waypoint deployment config
	ValuesTemplate *ValuesTemplate `hcl:"values_template,block"`

	AllowNoMatchingRelease bool   `hcl:"allow_no_matching_release,optional"`
	KubeContext            string `hcl:"kube_context,optional"`

	DiffContext int `hcl:"diff_context,optional"`
}

type ValuesTemplate struct {
	Data string `hcl:"data,optional"`
	Path string `hcl:"path,optional"`
}

type Platform struct {
	config PlatformConfig
}

// Config is a requisite to implement Configurable
func (p *Platform) Config() (interface{}, error) {
	return &p.config, nil
}

// ConfigSet is a requisite to implement ConfigurableNotify
func (p *Platform) ConfigSet(config interface{}) error {
	c, ok := config.(*PlatformConfig)
	if !ok {
		// The Waypoint SDK should ensure this never gets hit
		return fmt.Errorf("Expected *PlatformConfig")
	}

	if c.HelmfileBin != "" {
		if c.HelmfileVersion != "" {
			return errors.New("helmfile_bin and helmfile_version cannot be set concurrently")
		}
	}

	if c.HelmBin != "" {
		if c.HelmVersion != "" {
			return errors.New("helm_bin and helm_version cannot be set concurrently")
		}
	}

	data := c.ValuesTemplate.Data
	path := c.ValuesTemplate.Path
	if data == "" && path == "" {
		return errors.New("Either values_template.data or values_template.path needs to be set")
	}

	return nil
}

var _ component.Platform = &Platform{}

// DeployFunc is the requisite to implement component.Platform
func (p *Platform) DeployFunc() interface{} {
	// return a function which will be called by Waypoint
	return p.Deploy
}

func (p *Platform) Deploy(ctx context.Context, ui terminal.UI, src *component.Source, job *component.JobInfo, /*input *Input*/ image *docker.Image, deployConfig *component.DeploymentConfig) (*k8s.Deployment, error) {
	// We'll update the user in real time
	sg := ui.StepGroup()
	defer sg.Wait()

	// If we have a step set, abort it on exit
	var s terminal.Step
	defer func() {
		if s != nil {
			s.Abort()
		}
	}()

	id, err := component.Id()
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition,
			"component must have an ID: %v", err)
	}
	name := strings.ToLower(fmt.Sprintf("%s-%s", src.App, id))

	// Render templates if set
	s = sg.Add("Rendering values template...")

	// Build our template data
	var data tplData
	data.Env = deployConfig.Env()
	data.Workspace = job.Workspace
	//data.Populate(input)
	data.PopulateImage(image)

	// Render our template
	valuesPath, closer, err := p.renderTemplate(p.config.ValuesTemplate, &data)
	if closer != nil {
		defer closer()
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "rendering template: %v", err)
	}

	s.Done()
	s = sg.Add("Rendering Helmfile environment name...")

	var env string

	if p.config.EnvironmentTemplate != "" {
		tpl, err := template.New("environment_template").Parse(p.config.EnvironmentTemplate)
		if err != nil {
			return nil, status.Errorf(codes.FailedPrecondition, "parsing environment_template: %v", err)
		}

		var buf bytes.Buffer

		err = tpl.Execute(&buf, &data)
		if err != nil {
			return nil, status.Errorf(codes.FailedPrecondition, "executing environment_template: %v", err)
		}

		env = buf.String()
	}

	s.Done()
	s = sg.Add("Setting up Helmfile runtime configuration...")

	helmfileBin, helmBin, err := prepareBinaries(&p.config)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "failed fetching required binaries: %v", err)
	}

	// Add global flags

	args := []string{*helmfileBin, "-f", p.config.Path, "--state-values-file", valuesPath}

	if ns := p.config.Namespace; ns != "" {
		args = append(args, "-n", ns)
	}

	for _, s := range p.config.Selectors {
		args = append(args, "-l", s)
	}

	if c := p.config.KubeContext; c != "" {
		args = append(args, "--kube-context", c)
	}

	if p.config.AllowNoMatchingRelease {
		args = append(args, "--allow-no-matching-release")
	}

	if helmBin != nil && *helmBin != "" {
		args = append(args, "--helm-binary", *helmBin)
	}

	if env != "" {
		args = append(args, "-e", env)
	}

	// Add apply-specific flags

	args = append(args, "apply", "--detailed-exitcode", "--suppress-secrets")

	if c := p.config.DiffContext; c > 0 {
		args = append(args, "--context", strconv.Itoa(c))
	}

	s.Done()
	s = sg.Add("Executing command: %s", strings.Join(args, " "))

	//outputFile, err := os.Create(outputYAML)
	//if err != nil {
	//	u.Step(terminal.StatusError, fmt.Sprintf("Failed to create %s", outputYAML))
	//	return nil, err
	//}
	//defer outputFile.Close()

	var stderr bytes.Buffer
	helmfileCmd := exec.CommandContext(ctx,
		args[0], args[1:]...,
	)
	//helmfileCmd.Stdout = outputFile
	helmfileCmd.Stderr = &stderr
	helmfileCmd.Stdout = os.Stdout

	if err := helmfileCmd.Run(); err != nil {
		return nil, err
	}

	s.Update("Successfully finished running helmfile")
	s.Done()

	return &k8s.Deployment{
		Id:   id,
		Name: name,
	}, nil
}

func (p *Platform) renderTemplate(tpl *ValuesTemplate, data *tplData) (string, func(), error) {
	// Create a temporary directory to store our renders
	td, err := ioutil.TempDir("", "waypoint-helmfile")
	if err != nil {
		return "", nil, err
	}
	closer := func() {
		os.RemoveAll(td)
	}

	var (
		path string
		tmpl *template.Template
	)

	if tpl.Path != "" {
		fi, err := os.Stat(tpl.Path)
		if err != nil {
			return "", nil, status.Errorf(codes.FailedPrecondition, "calling stat on file specified by values_template.path: %v", err)
		}

		// Render
		if fi.IsDir() {
			return "", nil, status.Error(codes.FailedPrecondition, "reading file specified by values_template.path: must be a file, but was a directory")
		} else {
			// We'll copy the file into the temporary directory
			path = filepath.Join(td, filepath.Base(tpl.Path))

			// Build our template
			tmpl, err = template.New(filepath.Base(path)).ParseFiles(tpl.Path)
			if err != nil {
				return "", closer, status.Errorf(codes.InvalidArgument, "parsing file specified by values_template.path: %v", err)
			}
		}
	} else {
		tmpl, err = template.New("tmpl").Parse(tpl.Data)
		if err != nil {
			return "", closer, status.Errorf(codes.InvalidArgument, "parsing values_template.data: %v", err)
		}
	}

	// Create our target path
	f, err := os.Create(path)
	if err != nil {
		return "", closer, err
	}
	defer f.Close()

	return path, closer, tmpl.Execute(f, data)
}

var (
	_ component.Platform     = (*Platform)(nil)
	_ component.Configurable = (*Platform)(nil)
)
