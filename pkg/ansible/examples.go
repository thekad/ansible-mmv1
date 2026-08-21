// Copyright 2025 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0

package ansible

import (
	"fmt"
	"strings"

	mmv1api "github.com/GoogleCloudPlatform/magic-modules/mmv1/api"
	mmv1resource "github.com/GoogleCloudPlatform/magic-modules/mmv1/api/resource"
	"github.com/rs/zerolog/log"
)

type Examples struct {
	DocExamples  []*mmv1resource.Sample
	TestExamples []*mmv1resource.Sample
}

func NewExamplesFromMmv1(mmv1 *mmv1api.Resource) *Examples {
	docExamples := []*mmv1resource.Sample{}
	testExamples := []*mmv1resource.Sample{}
	for _, sample := range mmv1.Samples {
		if !sample.ExcludeBasicDoc {
			docExamples = append(docExamples, sample)
		}
		if !sample.ExcludeTest {
			testExamples = append(testExamples, sample)
		}
	}
	return &Examples{
		DocExamples:  docExamples,
		TestExamples: testExamples,
	}
}

// stepPhase derives the rendering phase from a step's name prefix.
func stepPhase(name string) string {
	switch {
	case strings.HasPrefix(name, "ansible_setup_"):
		return "setup"
	case strings.HasPrefix(name, "ansible_test_"):
		return "test"
	case strings.HasPrefix(name, "ansible_teardown_"):
		return "teardown"
	case strings.HasPrefix(name, "ansible_doc_"):
		return "doc"
	default:
		return "test"
	}
}

// ToString renders sample content for a given phase. Valid values for which:
//   - "doc":      all steps from DocExamples (no phase filtering)
//   - "setup":    ansible_setup_* steps from TestExamples
//   - "test":     ansible_test_* steps (and legacy unrecognized names) from TestExamples
//   - "teardown": ansible_teardown_* steps from TestExamples
//
// Steps within a sample are joined by a newline.
// Samples are joined by a #### separator line.
func (e *Examples) ToString(which string) string {
	separator := fmt.Sprintf("\n%s\n\n", strings.Repeat("#", 80))
	exampleStrings := []string{}

	var samples []*mmv1resource.Sample
	var useDocText bool

	switch which {
	case "doc":
		samples = e.DocExamples
		useDocText = true
	case "setup", "test", "teardown":
		samples = e.TestExamples
	}

	for _, sample := range samples {
		if len(sample.Steps) == 0 {
			log.Info().Msgf("skipping sample with no steps: %s", sample.Name)
			continue
		}
		stepStrings := []string{}
		for _, step := range sample.Steps {
			// For doc mode, include all steps without phase filtering.
			// For test phases, match by phase prefix.
			if which != "doc" && stepPhase(step.Name) != which {
				continue
			}
			var content string
			if useDocText {
				content = step.DocumentationHCLText
			} else {
				content = substituteTestVars(step.TestHCLText, step)
			}
			if len(content) <= 1 {
				log.Info().Msgf("skipping empty step: %s (sample: %s)", step.Name, sample.Name)
				continue
			}
			stepStrings = append(stepStrings, content)
		}
		if len(stepStrings) > 0 {
			exampleStrings = append(exampleStrings, strings.Join(stepStrings, "\n"))
		}
	}
	return strings.Join(exampleStrings, separator)
}

// substituteTestVars replaces MMv1 test-context placeholders (`%{key}`) with the
// step's literal `vars:` values. In the test-rendering pass MMv1 rewrites every
// `Vars` entry to a Terraform acctest placeholder (`%{key}`), expecting Terraform
// to substitute it from a context map at runtime. Ansible has no such mechanism,
// so we perform the substitution ourselves using the original literal values
// (which SetHCLText restores on the step after rendering). This lets a shared
// step template be parameterized per consumer (e.g. a unique resource-name
// suffix) while still emitting valid Ansible YAML.
func substituteTestVars(content string, step *mmv1resource.Step) string {
	for key, value := range step.Vars {
		content = strings.ReplaceAll(content, fmt.Sprintf("%%{%s}", key), value)
	}
	return content
}
