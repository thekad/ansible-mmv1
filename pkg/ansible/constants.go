// Copyright 2025 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0

package ansible

import (
	"strings"

	"github.com/GoogleCloudPlatform/magic-modules/mmv1/google"
)

// tfOnlyPropertyNames lists MMv1 property names that magic-modules adds for
// Terraform-only behavior and should be excluded from Ansible modules.
var tfOnlyPropertyNames = []string{
	"effectiveAnnotations",
	"effectiveLabels",
	"terraformLabels",
	"passwordWo",
	"passwordWoVersion",
}

// standardModuleRequirements are injected into every module DOCUMENTATION block.
var standardModuleRequirements = []string{
	"python >= 3.8",
	"requests >= 2.18.4",
	"google-auth >= 2.25.1",
}

// sensitiveFieldPatterns are substrings used to heuristically detect sensitive field names.
var sensitiveFieldPatterns = []string{
	"password",
	"passwd",
	"secret",
	"token",
	"key",
	"apikey",
	"api_key",
	"privatekey",
	"private_key",
	"credential",
	"auth",
	"authorization",
	"certificate",
	"cert",
}

// pythonKeywords are reserved words that must be quoted in generated Python.
var pythonKeywords = []string{
	"False", "None", "True", "and", "as", "assert", "break", "class", "continue",
	"def", "del", "elif", "else", "except", "finally", "for", "from", "global",
	"if", "import", "in", "is", "lambda", "nonlocal", "not", "or", "pass",
	"raise", "return", "try", "while", "with", "yield",
}

// reservedReturnKeys maps dict builtin method names to a safe alternative.
// ansible-test's validate-modules sanity check (bad-return-value-key) rejects
// any RETURN value key that collides with one of these (see
// FORBIDDEN_DICTIONARY_KEYS in ansible-test's validate_modules/constants.go),
// since such a key cannot be accessed with dot notation in Jinja. This
// substitution only applies when the property has no explicit api_name
// override in place.
var reservedReturnKeys = map[string]string{
	"clear":      "clear_value",
	"copy":       "copy_value",
	"fromkeys":   "fromkey_values",
	"get":        "get_value",
	"items":      "item_values",
	"keys":       "key_values",
	"pop":        "pop_value",
	"popitem":    "popitem_value",
	"setdefault": "setdefault_value",
	"update":     "update_value",
	"values":     "value_values",
}

func isTFOnlyPropertyName(name string) bool {
	for _, exclusion := range tfOnlyPropertyNames {
		// account for camel case and underscore versions
		if name == exclusion {
			return true
		}
		if name == google.Underscore(exclusion) {
			return true
		}
	}
	return false
}

func descriptionMentionsTFOnlyProperty(sentence string) bool {
	for _, name := range tfOnlyPropertyNames {
		if strings.Contains(sentence, google.Underscore(name)) {
			return true
		}
	}
	return false
}
