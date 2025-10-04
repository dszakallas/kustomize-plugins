package main

import (
	"fmt"
	"log"

	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resource"
	ktypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
	"sigs.k8s.io/kustomize/kyaml/resid"
	kyaml_utils "sigs.k8s.io/kustomize/kyaml/utils"
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
	valueNode := yaml.NewScalarRNode(sourceContent)

	items, err = applyReplacement(items, valueNode, r.Spec.Targets)
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
	Source  *SourceSpec       `yaml:"source,omitempty" json:"source,omitempty"`
	Targets []*TargetSelector `json:"targets,omitempty" yaml:"targets,omitempty"`
}

// TargetSelector defines the criteria for selecting and modifying target resources.
type TargetSelector struct {
	Select     *ktypes.Selector `yaml:"select" json:"select"`
	FieldPaths []string         `yaml:"fieldPaths" json:"fieldPaths"`
	Options    *FieldOptions    `yaml:"options,omitempty" json:"options,omitempty"`
}

// FieldOptions defines options for modifying fields in the target resources.
type FieldOptions struct {
	// Create the field if it does not exist.
	Create bool `json:"create,omitempty" yaml:"create,omitempty"`
}

type TargetSelectorRegex struct {
	targetSelector *TargetSelector
	selectRegex    *ktypes.SelectorRegex
}

func NewTargetSelectorRegex(ts *TargetSelector) (*TargetSelectorRegex, error) {
	tsr := new(TargetSelectorRegex)
	tsr.targetSelector = ts
	var err error

	tsr.selectRegex, err = ktypes.NewSelectorRegex(ts.Select)
	if err != nil {
		return nil, err
	}

	return tsr, nil
}

func (tsr *TargetSelectorRegex) Selects(id resid.ResId) bool {
	return tsr.selectRegex.MatchGvk(id.Gvk) && tsr.selectRegex.MatchName(id.Name) && tsr.selectRegex.MatchNamespace(id.Namespace)
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

func applyReplacement(nodes []*yaml.RNode, value *yaml.RNode, targetSelectors []*TargetSelector) ([]*yaml.RNode, error) {
	for _, selector := range targetSelectors {
		if selector.Select == nil {
			return nil, fmt.Errorf("target must specify resources to select")
		}
		if len(selector.FieldPaths) == 0 {
			selector.FieldPaths = []string{ktypes.DefaultReplacementFieldPath}
		}
		tsr, err := NewTargetSelectorRegex(selector)
		if err != nil {
			return nil, fmt.Errorf("error creating target selector: %w", err)
		}
		for _, possibleTarget := range nodes {
			id := makeResId(possibleTarget)

			// filter targets by label and annotation selectors
			selectByAnnoAndLabel, err := selectByAnnoAndLabel(possibleTarget, selector)
			if err != nil {
				return nil, err
			}
			if !selectByAnnoAndLabel {
				continue
			}

			if tsr.Selects(id) {
				err := copyValueToTarget(possibleTarget, value, selector)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return nodes, nil
}

func selectByAnnoAndLabel(n *yaml.RNode, t *TargetSelector) (bool, error) {
	if matchesSelect, err := matchesAnnoAndLabelSelector(n, t.Select); !matchesSelect || err != nil {
		return false, err
	}
	return true, nil
}

func matchesAnnoAndLabelSelector(n *yaml.RNode, selector *ktypes.Selector) (bool, error) {
	r := resource.Resource{RNode: *n}
	annoMatch, err := r.MatchesAnnotationSelector(selector.AnnotationSelector)
	if err != nil {
		return false, err
	}
	labelMatch, err := r.MatchesLabelSelector(selector.LabelSelector)
	if err != nil {
		return false, err
	}
	return annoMatch && labelMatch, nil
}

func makeResId(n *yaml.RNode) resid.ResId {
	apiVersion := n.Field(yaml.APIVersionField)
	var group, version string
	if apiVersion != nil {
		group, version = resid.ParseGroupVersion(yaml.GetValue(apiVersion.Value))
	}
	return resid.NewResIdWithNamespace(
		resid.Gvk{Group: group, Version: version, Kind: n.GetKind()}, n.GetName(), n.GetNamespace())
}

func copyValueToTarget(target *yaml.RNode, value *yaml.RNode, selector *TargetSelector) error {
	for _, fp := range selector.FieldPaths {
		createKind := yaml.Kind(0) // do not create
		if selector.Options != nil && selector.Options.Create {
			createKind = value.YNode().Kind
		}
		targetFieldList, err := target.Pipe(&yaml.PathMatcher{
			Path:   kyaml_utils.SmarterPathSplitter(fp, "."),
			Create: createKind})
		if err != nil {
			return errors.WrapPrefixf(err, "%s", fieldRetrievalError(fp, createKind != 0))
		}
		targetFields, err := targetFieldList.Elements()
		if err != nil {
			return errors.WrapPrefixf(err, "%s", fieldRetrievalError(fp, createKind != 0))
		}
		if len(targetFields) == 0 {
			return errors.Errorf("%s", fieldRetrievalError(fp, createKind != 0))
		}

		for _, t := range targetFields {
			if err := setFieldValue(t, value); err != nil {
				return err
			}
		}
	}
	return nil
}

func fieldRetrievalError(fieldPath string, isCreate bool) string {
	if isCreate {
		return fmt.Sprintf("unable to find or create field %q in replacement target", fieldPath)
	}
	return fmt.Sprintf("unable to find field %q in replacement target", fieldPath)
}

func setFieldValue(targetField *yaml.RNode, value *yaml.RNode) error {
	value = value.Copy()

	if targetField.YNode().Kind == yaml.ScalarNode {
		// For scalar, only copy the value (leave any type intact to auto-convert int->string or string->int)
		targetField.YNode().Value = value.YNode().Value
	} else {
		targetField.SetYNode(value.YNode())
	}

	return nil
}
