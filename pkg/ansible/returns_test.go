// Copyright 2025 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0

package ansible

import (
	"testing"

	mmv1api "github.com/GoogleCloudPlatform/magic-modules/mmv1/api"
)

func TestSafeReturnKey(t *testing.T) {
	cases := []struct {
		name    string
		apiName string
		want    string
	}{
		{"clear", "", "clear_value"},
		{"copy", "", "copy_value"},
		{"fromkeys", "", "fromkey_values"},
		{"get", "", "get_value"},
		{"items", "", "item_values"},
		{"keys", "", "key_values"},
		{"pop", "", "pop_value"},
		{"popitem", "", "popitem_value"},
		{"setdefault", "", "setdefault_value"},
		{"update", "", "update_value"},
		{"values", "", "value_values"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name+"_collides_with_dict_builtin", func(t *testing.T) {
			property := &mmv1api.Type{Name: tc.name, ApiName: tc.apiName}
			if got := safeReturnKey(property); got != tc.want {
				t.Fatalf("safeReturnKey() = %q, want %q", got, tc.want)
			}
		})
	}

	t.Run("does not rename when explicit api_name override is in place", func(t *testing.T) {
		property := &mmv1api.Type{Name: "items", ApiName: "customItems"}
		if got := safeReturnKey(property); got != "customItems" {
			t.Fatalf("safeReturnKey() = %q, want %q", got, "customItems")
		}
	})

	t.Run("passes through non-colliding names unchanged", func(t *testing.T) {
		property := &mmv1api.Type{Name: "displayName", ApiName: "displayName"}
		if got := safeReturnKey(property); got != "displayName" {
			t.Fatalf("safeReturnKey() = %q, want %q", got, "displayName")
		}
	})
}
