# kustomize-plugins

## Plugins

- [ResourceInjector](#resourceinjector) - Render a Kustomize source and inject the resulting YAML into specified
  fields
- [YqTransform](#yqtransform) - Apply yq expressions to transform specific fields in Kubernetes resources

## ResourceInjector

The `ResourceInjector` is a Kustomize plugin designed to render a Kustomize source and inject the resulting YAML
into specified fields of other resources. This allows for dynamic configuration and management of Kubernetes resources
by embedding the output of one Kustomize build into others.

### How It Works (ResourceInjector)

The plugin operates as a Kustomize function and is configured through a custom resource definition (CRD) within your
Kustomize setup. It performs the following steps:

1. **Renders a Kustomize Source**: It takes a path to a Kustomize source directory, builds it, and captures the
   resulting YAML output.
2. **Selects Target Resources**: It identifies target resources within the Kustomize build using a selector based on
   properties like `apiVersion`, `kind`, and `name`.
3. **Injects Content**: It injects the rendered YAML from the source as a string into one or more specified fields of
   the selected target resources.

### Configuration (ResourceInjector)

The `ResourceInjector` is configured using a YAML file that defines the `source` to be rendered and the `targets` for
injection.

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

### Fields (ResourceInjector)

- `spec.source.path`: The path to the Kustomize source directory to be rendered. This path is relative to the
  `kustomization.yaml` file that includes the plugin.
- `spec.source.fieldPath`: Optionally specify a field in the YAML to project.
- `spec.targets`: A list of target selectors to identify where the rendered content should be injected.
- `spec.targets.select`: A selector to identify the target resources. It supports fields like `group`, `version`,
  `kind`, `name`, and `namespace`.
- `spec.targets.fieldPaths`: A list of fields in the target resources where the rendered YAML should be injected. The
  content is injected as a string.
- `spec.targets.options.create`: (Optional) A boolean that, if `true`, creates the specified field if it does not
  already exist in the target resource.

### Usage (ResourceInjector)

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

1. Build the Kustomize source located at `../common-resources`.
2. Find the `ConfigMap` named `my-app-configmap`.
3. Inject the rendered YAML from `../common-resources` into the `data.injected-config` field of the `ConfigMap`.

## YqTransform

The `YqTransform` is a Kustomize plugin designed to apply [yq](https://github.com/mikefarah/yq) expressions to
transform specific fields within Kubernetes resources. This allows for powerful in-place transformations such as
sorting arrays, filtering values, modifying nested structures, and more.

### How It Works (YqTransform)

The plugin operates as a Kustomize function and is configured through a custom resource definition (CRD) within your
Kustomize setup. It performs the following steps:

1. **Selects Target Fields**: It identifies specific fields within target resources using a selector based on
   properties like `apiVersion`, `kind`, `name`, and field paths.
2. **Applies yq Expression**: It applies the specified yq expression to transform each selected field in place.
3. **Updates Resources**: The transformed fields are written back to the resources, preserving the rest of the resource
   structure.

### Configuration (YqTransform)

The `YqTransform` is configured using a YAML file that defines the `expression` to apply and the `targets` to select.

Here is an example of a `YqTransform` configuration:

```yaml
apiVersion: kustomize-plugins.dszakallas.github.com/v1alpha1
kind: YqTransform
metadata:
  name: sort-env-vars
  annotations:
    config.kubernetes.io/function: |
      exec:
        path: kustomize-plugin-yqtransform
spec:
  expression: "sort_by(.name)"
  targets:
    - select:
        kind: Deployment
      fieldPaths:
        - spec.template.spec.containers.*.env
```

### Fields (YqTransform)

- `spec.expression`: A yq expression to apply to the selected fields. The expression operates on each selected field
  independently.
- `spec.targets`: A list of target selectors to identify which fields should be transformed.
- `spec.targets.select`: A selector to identify the target resources. It supports fields like `group`, `version`,
  `kind`, `name`, and `namespace`.
- `spec.targets.fieldPaths`: A list of field paths within the target resources to transform. Supports array wildcards
  like `[]` to apply the transformation to all array elements.
- `spec.targets.options.create`: (Optional) A boolean that, if `true`, creates the specified field if it does not
  already exist in the target resource.

### Usage (YqTransform)

To use the `YqTransform` plugin, you need to include it in your `kustomization.yaml` as a transformer.

Here is an example of how to use it in a `kustomization.yaml`:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - deployment.yaml

transformers:
  - |-
    apiVersion: kustomize-plugins.dszakallas.github.com/v1alpha1
    kind: YqTransform
    metadata:
      name: sort-env-vars
      annotations:
        config.kubernetes.io/function: |
          exec:
            path: kustomize-plugin-yqtransform
    spec:
      expression: "sort_by(.name)"
      targets:
        - select:
            kind: Deployment
            name: my-app
          fieldPaths:
            - spec.template.spec.containers.*.env
```

In this example, the `YqTransform` will:

1. Find the `Deployment` named `my-app`.
2. Locate all `env` arrays within the deployment's container specifications.
3. Apply the `sort_by(.name)` yq expression to sort environment variables by name in each container.

### Common Use Cases

**Sort environment variables:**

```yaml
expression: "sort_by(.name)"
fieldPaths:
  - spec.template.spec.containers.*.env
```

**Filter items from an array:**

```yaml
expression: "map(select(.name != \"DEBUG\"))"
fieldPaths:
  - spec.template.spec.containers.*.env
```

**Add or modify fields:**

```yaml
expression: ". + {\"imagePullPolicy\": \"Always\"}"
fieldPaths:
  - spec.template.spec.containers.*
```

**Transform nested structures:**

```yaml
expression: ".limits.memory = \"2Gi\""
fieldPaths:
  - spec.template.spec.containers.*.resources
```
