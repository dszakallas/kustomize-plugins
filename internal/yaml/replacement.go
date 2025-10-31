package yaml

import (
	"fmt"

	"sigs.k8s.io/kustomize/api/resource"
	ktypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/resid"
	kyaml_utils "sigs.k8s.io/kustomize/kyaml/utils"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// Transform defines an interface for applying transformations to YAML nodes.
type Transform interface {
	CreateKind() yaml.Kind
	Apply(target *yaml.RNode) error
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

func ApplyTransform(transform Transform, nodes []*yaml.RNode, targetSelectors []*TargetSelector) ([]*yaml.RNode, error) {
	for _, selector := range targetSelectors {
		if selector.Select == nil {
			return nil, fmt.Errorf("target must specify resources to select")
		}
		if len(selector.FieldPaths) == 0 {
			selector.FieldPaths = []string{ktypes.DefaultReplacementFieldPath}
		}
		tsr, err := newTargetSelectorRegex(selector)
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
				err := applyTransformToTarget(transform, possibleTarget, selector)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return nodes, nil
}

type targetSelectorRegex struct {
	targetSelector *TargetSelector
	selectRegex    *ktypes.SelectorRegex
}

func newTargetSelectorRegex(ts *TargetSelector) (*targetSelectorRegex, error) {
	tsr := new(targetSelectorRegex)
	tsr.targetSelector = ts
	var err error

	tsr.selectRegex, err = ktypes.NewSelectorRegex(ts.Select)
	if err != nil {
		return nil, err
	}

	return tsr, nil
}

func (tsr *targetSelectorRegex) Selects(id resid.ResId) bool {
	return tsr.selectRegex.MatchGvk(id.Gvk) && tsr.selectRegex.MatchName(id.Name) && tsr.selectRegex.MatchNamespace(id.Namespace)
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

func applyTransformToTarget(transform Transform, target *yaml.RNode, selector *TargetSelector) error {
	for _, fp := range selector.FieldPaths {
		createKind := yaml.Kind(0) // do not create
		if selector.Options != nil && selector.Options.Create {
			createKind = transform.CreateKind()
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
			if err := transform.Apply(t); err != nil {
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
