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

func (e *Examples) ToString(which string) string {
	separator := fmt.Sprintf("\n%s\n\n", strings.Repeat("#", 80))
	exampleStrings := []string{}
	samples := []*mmv1resource.Sample{}
	switch which {
	case "doc":
		samples = e.DocExamples
	case "test":
		samples = e.TestExamples
	}
	for _, sample := range samples {
		// Use only the first step as the canonical create example
		if len(sample.Steps) == 0 {
			log.Info().Msgf("skipping sample with no steps: %s", sample.Name)
			continue
		}
		step := sample.Steps[0]
		var content string
		if which == "doc" {
			content = step.DocumentationHCLText
		} else {
			content = step.TestHCLText
		}
		if len(content) <= 1 {
			log.Info().Msgf("skipping empty sample: %s", sample.Name)
			continue
		}
		exampleStrings = append(exampleStrings, content)
	}
	return strings.Join(exampleStrings, separator)
}
