// Copyright 2025 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0

package ansible

import (
	"strings"
	"testing"

	mmv1api "github.com/GoogleCloudPlatform/magic-modules/mmv1/api"
)

// TestArgSpecMapOfNestedObjectsRendersAsList guards against a regression of a
// real runtime bug: modeling a Map as a native dict with "options=" in the
// argument_spec caused Ansible's own argument validation to expect the
// suboptions directly under the field (e.g. "missing required arguments:
// enforcement_mode, evaluation_mode found in cluster_admission_rules")
// instead of nested under each arbitrary map key. Following the upstream
// Terraform provider's convention, a Map must render as type="list",
// elements="dict", with the value_type's properties (plus an injected key
// field) as per-item "options=", which Ansible validates correctly.
func TestArgSpecMapOfNestedObjectsRendersAsList(t *testing.T) {
	property := &mmv1api.Type{
		Name:    "clusterAdmissionRules",
		Type:    "Map",
		KeyName: "cluster",
		ValueType: &mmv1api.Type{
			Type: "NestedObject",
			Properties: []*mmv1api.Type{
				{Name: "evaluationMode", Type: "Enum", Required: true, EnumValues: []string{"ALWAYS_ALLOW", "REQUIRE_ATTESTATION", "ALWAYS_DENY"}},
				{Name: "enforcementMode", Type: "Enum", Required: true},
			},
		},
	}

	options := convertPropertiesToOptions([]*mmv1api.Type{property}, nil, false, true)
	argSpec := NewArgSpecFromOptions(options, nil)
	got := argSpec.ToString()

	if !strings.Contains(got, `type="list"`) {
		t.Fatalf("argspec for Map field should declare type=list, got:\n%s", got)
	}
	if !strings.Contains(got, `elements="dict"`) {
		t.Fatalf("argspec for Map field should declare elements=dict, got:\n%s", got)
	}
	// The critical regression check: options= must NOT be a dict of the
	// suboptions applied directly to the field itself (that's the dict-based
	// bug); it must be nested inside a per-item "options=dict(...)" block.
	if strings.Contains(got, `type="dict"`) {
		t.Fatalf("argspec for Map field must not render as type=dict, got:\n%s", got)
	}
	if !strings.Contains(got, "cluster=dict(") {
		t.Fatalf("argspec must include the injected key field 'cluster', got:\n%s", got)
	}
	if !strings.Contains(got, "evaluation_mode=dict(") {
		t.Fatalf("argspec must include value_type property 'evaluation_mode', got:\n%s", got)
	}
}

func TestArgSpecMapWithScalarValueTypeHasNoOptions(t *testing.T) {
	property := &mmv1api.Type{
		Name:      "labels",
		Type:      "Map",
		ValueType: &mmv1api.Type{Type: "String"},
	}

	options := convertPropertiesToOptions([]*mmv1api.Type{property}, nil, false, true)
	argSpec := NewArgSpecFromOptions(options, nil)
	got := argSpec.ToString()

	if !strings.Contains(got, `type="list"`) {
		t.Fatalf("argspec for scalar-valued Map field should declare type=list, got:\n%s", got)
	}
	if strings.Contains(got, "options=dict(") {
		t.Fatalf("argspec for scalar-valued Map field should have no options=, got:\n%s", got)
	}
}
