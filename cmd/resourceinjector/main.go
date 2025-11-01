package main

import (
	"fmt"
	"log"

	"gitops.szakallas.eu/plugins/internal/transform"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func main() {
	api := &API{}

	// Use the kyaml framework to build a command-line tool.
	cmd := command.Build(
		framework.SimpleProcessor{Config: api, Filter: api},
		command.StandaloneEnabled,
		false,
	)
	command.AddGenerateDockerfile(cmd)

	if err := cmd.Execute(); err != nil {
		log.Fatalf("Error executing command: %v", err)
	}
}

// API is the top-level configuration for the function.
type API struct {
	Metadata struct {
		// Name is the Deployment Resource and Container name
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec ResourceInjectorSpec `yaml:"spec" json:"spec"`
}

type setValue struct {
	Value *yaml.RNode
}

func (s *setValue) CreateKind() yaml.Kind {
	return s.Value.YNode().Kind
}

func (s *setValue) Apply(target *yaml.RNode) error {
	value := s.Value.Copy()

	if target.YNode().Kind == yaml.ScalarNode {
		// For scalar, only copy the value (leave any type intact to auto-convert int->string or string->int)
		target.YNode().Value = value.YNode().Value
	} else {
		target.SetYNode(value.YNode())
	}

	return nil
}

// Filter reads the source, builds it if necessary, and injects the result
// into the target resources.
func (r *API) Filter(items []*yaml.RNode) ([]*yaml.RNode, error) {
	if r.Spec.Source == nil || r.Spec.Source.Path == "" {
		return nil, fmt.Errorf("source.path must be specified")
	}

	// 1. Render the source content.
	source, err := kustomizeSource(r.Spec.Source.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to render source: %w", err)
	}

	if r.Spec.Source.FieldPath != "" {
		var err error
		source, err = source.Pipe(yaml.Lookup(r.Spec.Source.FieldPath))
		if err != nil {
			return nil, fmt.Errorf("failed to lookup field path in rendered source: %w", err)
		}
		if source == nil {
			return nil, fmt.Errorf("field path %q not found in rendered source", r.Spec.Source.FieldPath)
		}
	}

	// We wrap it in a string node as the value needs to be injected as a string.
	sourceContent, err := source.String()
	if err != nil {
		return nil, fmt.Errorf("failed to convert source to string: %w", err)
	}
	setter := setValue{Value: yaml.NewScalarRNode(sourceContent)}

	items, err = transform.Apply(&setter, items, r.Spec.Targets)
	if err != nil {
		return nil, fmt.Errorf("failed to apply replacements: %w", err)
	}

	return items, nil
}

// SourceSpec defines the source of the content to be injected.
type SourceSpec struct {
	// Path to the kustomization directory.
	Path string `yaml:"path" json:"path"`
	// Optional field path to extract from the rendered source.
	FieldPath string `yaml:"fieldPath,omitempty" json:"fieldPath,omitempty"`
}

// ResourceInjectorSpec defines the configuration for the resource injector.
type ResourceInjectorSpec struct {
	Source  *SourceSpec                 `yaml:"source,omitempty" json:"source,omitempty"`
	Targets []*transform.TargetSelector `json:"targets,omitempty" yaml:"targets,omitempty"`
}

// kustomizeSource reads a path and, if it's a kustomization directory, builds it.
// Otherwise, it reads the content of the file. It returns the content as a structured yaml node.
func kustomizeSource(sourcePath string) (*yaml.RNode, error) {
	fSys := filesys.MakeFsOnDisk()

	// Check if the path is a directory
	if fSys.IsDir(sourcePath) {
		// Treat as a kustomization directory and build it.
		k := krusty.MakeKustomizer(krusty.MakeDefaultOptions())
		resMap, err := k.Run(fSys, sourcePath)
		if err != nil {
			return nil, fmt.Errorf("kustomize build failed for %q: %w", sourcePath, err)
		}
		yamlBytes, err := resMap.AsYaml()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal kustomize output to YAML: %w", err)
		}
		return yaml.Parse(string(yamlBytes))
	}

	// If not a kustomization, treat it as a plain file.
	content, err := fSys.ReadFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read source file %q: %w", sourcePath, err)
	}

	return yaml.Parse(string(content))
}
