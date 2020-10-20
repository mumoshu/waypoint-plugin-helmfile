# Waypoint Helmfile Plugin

`waypoint-plugin-helmfile` is a [Waypoint](https://github.com/hashicorp/waypoint) plugin to trigger [Helmfile](https://github.com/roboll/helmfile) deployments.

The implementation is conceptually very similar to the builtin [Exec](https://www.waypointproject.io/plugins/exec) plugin as this plugin shells out to run `Helmfile`.

You may prefer this over [`Exec`](https://www.waypointproject.io/plugins/exec) when you want:

- Ability to install required versions of Helmfile and Helm on deploy
- Standardized usage of Helmfile in Waypoint

## Configuration

```hcl
project = "myapp"

app "myapp" {
  deploy {
    use "helmfile" {
      // The semver constraint for the version of Helmfile to install and use.
      // For example, ">= 0.132.0" instructs the plugin to install the latest Helmfile version that is greater than or equal to 0.132.0
      //
      // This feature is backed by the package manager [shoal](https://github.com/mumoshu/shoal)
      // is able to fetch any packages hosted in https://github.com/fishworks/fish-food
      helmfile_version = ""

      // The semver constraint for the version of Helm. E.g. "3.3.4" or ">= 3.3.4"
      helm_version = ""

      // The path to the helmfile executable. Defaults to "helmfile" and conflicts with `helmfile_version`
      helmfile_bin = ""

      // The path to the helm executable. Defaults to "helm" and conflicts with `helm_version`
      helm_bin = ""

      // Corresponds to `helmfile -e <WAYPOINT_WORKSPACE>`
      environment_template = "{{ .Workspace }}"

      // Corresponds to `helmfile --state-values-file path/to/tmp.yaml`
      // where tmp.yaml is rendered from the following go template.
      values_template {
        data = <<-EOS
        image:
          repository: {{.Input.DockerImageName}}
          tag: {{.Input.DockerImageTag}}
        env:
        {{- range $k, $v := .Env }}
          {{ $k }}: {{ $v }}
        {{- end }}
        EOS
      }

      // Path to the helmfile config. Maps to `helmfile -f <PATH>`
      path = "helmfile.yaml"
 
      // Path to the working directory of the Helmfile process
      dir = "."

      // Maps to `helmfile --selector foo=bar`
      selectors = ["foo=bar"]

      // The default Kubernetes namespace passed to Helmfile.
      // Maps to `helmfile -n NAMESPACE` 
      namespace = ""

      // The default kubeconfig context passed to Helmfile.
      // Maps to `helmfile --kube-context CONTEXT`
      kube_context = ""

      // Maps to `helmfile --allow-no-matching-release`
      allow_no_matching_release = false

      // The number of lines in the context around changes to output
      // Maps to `helmfile apply --contect N`
      diff_context = 3
    }
  }
}
```

`values_template` is important to pass required environment variables to your application.

Waypoint, especially commands like `waypoint logs`, requires that the application's container image to be built by
`waypoint` so that the image has the "waypoint entrypoint" embedded.

Within `values_template`, you can use the same set of template parameters as the Exec plugin.
Please refer to the [Templates](https://www.waypointproject.io/plugins/exec#templates) section of the Exec plugin documentato for more information.

## Install

To install the plugin, run the following command:

```bash
$ TARGET=path/to/your/waypoint/project make install
```

The plugin binary gets installed into:

```
path/to/your/waypoint/project/.config/waypoint/plugins/waypoint-plugin-helmfile
```

so that the `waypoint` command can automatically discover the binary.

## Deployment steps

1. `waypoint init`
2. Update `waypoint.hcl` with:
  ```hcl
  deploy {

    use "helmfile" {
      // Corresponds to `helmfile -e <WAYPOINT_WORKSPACE>`
      environment_template = "{{ .Workspace }}"

      // Corresponds to `helmfile --state-values-file path/to/tmp.yaml`
      // where tmp.yaml is rendered from the following go template.
      values_template {
        data = <<-EOS
        image:
          repository: {{.Input.DockerImageName}}
          tag: {{.Input.DockerImageTag}}
        env:
        {{- range $k, $v := .Env }}
          {{ $k }}: {{ $v }}
        {{- end }}
        EOS
      }
    }
  }
  ```
3. `waypoint init` to verify the configuration
4. `waypoint up` to build and deploy the app
5. Validate that the app is available at the Deployment URL
6. Validate that k8s resources were deployed: `kubectl get deploy` and `kubectl get svc`  

## Cleanup

1. `waypoint destroy`
