package main

import (
	"fmt"
	"log"

	"github.com/mikefarah/yq/v4/pkg/yqlib"
	internalyaml "gitops.szakallas.eu/plugins/internal/yaml"
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

// YqTransformSpec defines the configuration for the yq transformer.
type YqTransformSpec struct {
	Expression string                         `yaml:"expression" json:"expression"`
	Targets    []*internalyaml.TargetSelector `json:"targets,omitempty" yaml:"targets,omitempty"`
}

// API is the top-level configuration for the function.
type API struct {
	Metadata struct {
		// Name is the Deployment Resource and Container name
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec YqTransformSpec `yaml:"spec" json:"spec"`
}

// Filter applies the yq expression to the target resources.
func (r *API) Filter(items []*yaml.RNode) ([]*yaml.RNode, error) {
	if r.Spec.Expression == "" {
		return nil, fmt.Errorf("expression must be specified")
	}

	yq := &yqTransform{
		Expression: r.Spec.Expression,
		Evaluator:  yqlib.NewAllAtOnceEvaluator(),
	}

	items, err := internalyaml.ApplyTransform(yq, items, r.Spec.Targets)
	if err != nil {
		return nil, fmt.Errorf("failed to apply yq: %w", err)
	}

	return items, nil
}

type yqTransform struct {
	Expression string
	Evaluator  yqlib.Evaluator
}

func (s *yqTransform) CreateKind() yaml.Kind {
	return yaml.Kind(0) // Cannot create a new node
}

func (s *yqTransform) Apply(target *yaml.RNode) error {
	var inputNode yqlib.CandidateNode
	if err := inputNode.UnmarshalYAML(target.YNode(), nil); err != nil {
		return fmt.Errorf("failed to unmarshal node: %w", err)
	}

	// Evaluate the expression
	result, err := s.Evaluator.EvaluateNodes(s.Expression, &inputNode)
	if err != nil {
		return fmt.Errorf("failed to evaluate expression: %w", err)
	}

	if result.Len() == 0 {
		return fmt.Errorf("expression produced no results")
	}

	// Get the first result from the list
	firstResult := result.Front()
	if firstResult == nil {
		return fmt.Errorf("expression produced empty result")
	}

	resultCandidate, ok := firstResult.Value.(*yqlib.CandidateNode)
	if !ok {
		return fmt.Errorf("unexpected result type")
	}

	outNode, err := resultCandidate.MarshalYAML()
	if err != nil {
		return fmt.Errorf("failed to marshal yq result: %w", err)
	}

	// Replace the target node's content with the transformed content
	target.SetYNode(outNode)

	return nil
}
