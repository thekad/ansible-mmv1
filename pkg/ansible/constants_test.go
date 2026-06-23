// Copyright 2025 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0

package ansible

import "testing"

func TestIsTFOnlyPropertyName(t *testing.T) {
	// Every name in tfOnlyPropertyNames must match both its camelCase original
	// and the snake_case equivalent produced by google.Underscore().
	camelCases := []struct {
		name      string
		snakeCase string
	}{
		{"effectiveAnnotations", "effective_annotations"},
		{"effectiveLabels", "effective_labels"},
		{"terraformLabels", "terraform_labels"},
		{"passwordWo", "password_wo"},
		{"passwordWoVersion", "password_wo_version"},
	}

	for _, tc := range camelCases {
		tc := tc
		t.Run(tc.name+"_camelCase_is_TF_only", func(t *testing.T) {
			if !isTFOnlyPropertyName(tc.name) {
				t.Fatalf("isTFOnlyPropertyName(%q) = false, want true", tc.name)
			}
		})
		t.Run(tc.snakeCase+"_snakeCase_is_TF_only", func(t *testing.T) {
			if !isTFOnlyPropertyName(tc.snakeCase) {
				t.Fatalf("isTFOnlyPropertyName(%q) = false, want true", tc.snakeCase)
			}
		})
	}

	t.Run("empty string is not TF-only", func(t *testing.T) {
		if isTFOnlyPropertyName("") {
			t.Fatal("isTFOnlyPropertyName(\"\") = true, want false")
		}
	})

	ordinary := []string{
		"labels", "annotations", "name", "description",
		"project", "region", "location", "displayName",
	}
	for _, name := range ordinary {
		name := name
		t.Run(name+"_is_not_TF_only", func(t *testing.T) {
			if isTFOnlyPropertyName(name) {
				t.Fatalf("isTFOnlyPropertyName(%q) = true, want false", name)
			}
		})
	}

	t.Run("partial prefix match is not TF-only", func(t *testing.T) {
		// "effective" alone is not a TF-only name
		if isTFOnlyPropertyName("effective") {
			t.Fatal("isTFOnlyPropertyName(\"effective\") = true, want false")
		}
	})
}

func TestDescriptionMentionsTFOnlyProperty(t *testing.T) {
	// Each snake_case form of a TF-only name should trigger a match.
	triggering := []struct {
		sentence string
		reason   string
	}{
		{"Managed by effective_labels field.", "effective_labels"},
		{"See effective_annotations for details.", "effective_annotations"},
		{"The terraform_labels key is reserved.", "terraform_labels"},
		{"Use password_wo to supply write-only password.", "password_wo"},
		{"Tracks the password_wo_version value.", "password_wo_version"},
		// Substring within a longer word still matches because Contains is used.
		{"Seealso:effective_labels.", "effective_labels embedded without spaces"},
	}

	for _, tc := range triggering {
		tc := tc
		t.Run("detects_"+tc.reason, func(t *testing.T) {
			if !descriptionMentionsTFOnlyProperty(tc.sentence) {
				t.Fatalf("descriptionMentionsTFOnlyProperty(%q) = false, want true", tc.sentence)
			}
		})
	}

	clean := []string{
		"The name of the resource.",
		"A human-readable display name.",
		"Region where the resource resides.",
		"Labels attached to the resource.",      // "labels" alone, not "effective_labels"
		"Annotations on the resource.",           // "annotations" alone
		"The password field for authentication.", // "password" alone, not "password_wo"
	}
	for _, sentence := range clean {
		sentence := sentence
		t.Run("no_match_for: "+sentence[:min(len(sentence), 30)], func(t *testing.T) {
			if descriptionMentionsTFOnlyProperty(sentence) {
				t.Fatalf("descriptionMentionsTFOnlyProperty(%q) = true, want false", sentence)
			}
		})
	}

	t.Run("empty sentence returns false", func(t *testing.T) {
		if descriptionMentionsTFOnlyProperty("") {
			t.Fatal("descriptionMentionsTFOnlyProperty(\"\") = true, want false")
		}
	})
}


