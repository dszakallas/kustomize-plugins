# kustomize-plugins

## ResourceInjector

The `ResourceInjector` is a Kustomize plugin designed to render a Kustomize source and inject the resulting YAML into specified fields of other resources. This allows for dynamic configuration and management of Kubernetes resources by embedding the output of one Kustomize build into others.

### How It Works

The plugin operates as a Kustomize function and is configured through a custom resource definition (CRD) within your Kustomize setup. It performs the following steps:

1.  **Renders a Kustomize Source**: It takes a path to a Kustomize source directory, builds it, and captures the resulting YAML output.
2.  **Selects Target Resources**: It identifies target resources within the Kustomize build using a selector based on properties like `apiVersion`, `kind`, and `name`.
3.  **Injects Content**: It injects the rendered YAML from the source as a string into one or more specified fields of the selected target resources.

### Configuration

The `ResourceInjector` is configured using a YAML file that defines the `source` to be rendered and the `targets` for injection.

Here is an example of a `ResourceInjector` configuration:

```yaml
apiVersion: kustomize-plugins.dszakallas.github.com/v1alpha1
kind: ResourceInjector
metadata:
  name: inject
  annotations:
    config.kubernetes.io/function: |
      exec:
        path: kustomize-plugin-resourceinjector
spec:
  source:
    path: ../path/to/source/kustomization
    fieldPath: spec
  targets:
    - select:
        kind: ConfigMap
        name: my-configmap
      fieldPaths:
        - data.mykey
```

### Fields

*   `spec.source.path`: The path to the Kustomize source directory to be rendered. This path is relative to the `kustomization.yaml` file that includes the plugin.
*   `spec.source.fieldPath`: Optionally specify a field in the YAML to project.
*   `spec.targets`: A list of target selectors to identify where the rendered content should be injected.
*   `spec.targets.select`: A selector to identify the target resources. It supports fields like `group`, `version`, `kind`, `name`, and `namespace`.
*   `spec.targets.fieldPaths`: A list of fields in the target resources where the rendered YAML should be injected. The content is injected as a string.
*   `spec.targets.options.create`: (Optional) A boolean that, if `true`, creates the specified field if it does not already exist in the target resource.

### Usage

To use the `ResourceInjector` plugin, you need to include it in your `kustomization.yaml` as a generator or transformer.

Here is an example of how to use it in a `kustomization.yaml`:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - my-app-deployment.yaml
  - my-app-configmap.yaml

transformers:
  - |-
    apiVersion: kustomize-plugins.dszakallas.github.com/v1alpha1
    kind: ResourceInjector
    metadata:
      name: inject
      annotations:
        config.kubernetes.io/function: |
          exec:
            path: kustomize-plugin-resourceinjector
    spec:
      source:
        path: ../common-resources
      targets:
        - select:
            kind: ConfigMap
            name: my-app-configmap
          fieldPaths:
            - data.injected-config
```

In this example, the `ResourceInjector` will:

1.  Build the Kustomize source located at `../common-resources`.
2.  Find the `ConfigMap` named `my-app-configmap`.
3.  Inject the rendered YAML from `../common-resources` into the `data.injected-config` field of the `ConfigMap`.
