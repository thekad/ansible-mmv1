// Copyright 2025 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0

package ansible

import (
	"reflect"
	"testing"

	mmv1api "github.com/GoogleCloudPlatform/magic-modules/mmv1/api"
)

func TestGetMutuallyExclusive(t *testing.T) {
	t.Run("dedupes circular conflicts", func(t *testing.T) {
		options := map[string]*Option{
			"a": {Name: "a", Conflicts: []string{"b", "c"}},
			"b": {Name: "b", Conflicts: []string{"c", "a"}},
			"c": {Name: "c", Conflicts: []string{"a", "b"}},
		}

		got := translateMmv1Conflicts(options)
		want := [][]string{{"a", "b", "c"}}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("getMutuallyExclusive() = %#v, want %#v", got, want)
		}
	})

	t.Run("returns nil when no conflicts", func(t *testing.T) {
		options := map[string]*Option{
			"a": {Name: "a"},
			"b": {Name: "b"},
		}

		if got := translateMmv1Conflicts(options); got != nil {
			t.Fatalf("getMutuallyExclusive() = %#v, want nil", got)
		}
	})

	t.Run("normalizes dotted conflict names", func(t *testing.T) {
		options := map[string]*Option{
			"fooBar": {Name: "fooBar", Conflicts: []string{"parent.child"}},
		}

		got := translateMmv1Conflicts(options)
		want := [][]string{{"child", "foo_bar"}}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("getMutuallyExclusive() = %#v, want %#v", got, want)
		}
	})

	t.Run("drops group when only TF-only conflicts remain", func(t *testing.T) {
		options := map[string]*Option{
			"foo": {Name: "foo", Conflicts: []string{"effectiveLabels", "terraformLabels"}},
		}

		if got := translateMmv1Conflicts(options); got != nil {
			t.Fatalf("getMutuallyExclusive() = %#v, want nil", got)
		}
	})

	t.Run("keeps group when TF-only conflicts are stripped but normal ones remain", func(t *testing.T) {
		options := map[string]*Option{
			"foo": {Name: "foo", Conflicts: []string{"effectiveLabels", "terraformLabels", "bar"}},
		}

		got := translateMmv1Conflicts(options)
		want := [][]string{{"bar", "foo"}}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("getMutuallyExclusive() = %#v, want %#v", got, want)
		}
	})
}

func TestGetRequiredOneOf(t *testing.T) {
	t.Run("dedupes circular at-least-one-of groups", func(t *testing.T) {
		// AtLeastOneOf already includes the owning option, so each entry lists the
		// full group. All three produce the same sorted key and should collapse to one.
		options := map[string]*Option{
			"a": {Name: "a", AtLeastOneOf: []string{"a", "b", "c"}},
			"b": {Name: "b", AtLeastOneOf: []string{"a", "b", "c"}},
			"c": {Name: "c", AtLeastOneOf: []string{"a", "b", "c"}},
		}

		got := translateMmv1AtLeastOneOf(options)
		want := [][]string{{"a", "b", "c"}}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("getRequiredOneOf() = %#v, want %#v", got, want)
		}
	})

	t.Run("returns nil when no at-least-one-of", func(t *testing.T) {
		options := map[string]*Option{
			"a": {Name: "a"},
			"b": {Name: "b"},
		}

		if got := translateMmv1AtLeastOneOf(options); got != nil {
			t.Fatalf("getRequiredOneOf() = %#v, want nil", got)
		}
	})

	t.Run("normalizes dotted at-least-one-of names", func(t *testing.T) {
		options := map[string]*Option{
			"foo": {Name: "foo", AtLeastOneOf: []string{"parent.foo", "parent.bar"}},
		}

		got := translateMmv1AtLeastOneOf(options)
		want := [][]string{{"bar", "foo"}}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("getRequiredOneOf() = %#v, want %#v", got, want)
		}
	})

	t.Run("drops group when only TF-only entries remain", func(t *testing.T) {
		options := map[string]*Option{
			"foo": {Name: "foo", AtLeastOneOf: []string{"foo", "effectiveLabels"}},
		}

		// after filtering effectiveLabels, only "foo" survives - group < 2, must be nil
		if got := translateMmv1AtLeastOneOf(options); got != nil {
			t.Fatalf("getRequiredOneOf() = %#v, want nil", got)
		}
	})

	t.Run("keeps group when TF-only entries are stripped but normal ones remain", func(t *testing.T) {
		options := map[string]*Option{
			"foo": {Name: "foo", AtLeastOneOf: []string{"foo", "effectiveLabels", "bar"}},
		}

		got := translateMmv1AtLeastOneOf(options)
		want := [][]string{{"bar", "foo"}}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("getRequiredOneOf() = %#v, want %#v", got, want)
		}
	})
}

func TestGetRequiredTogether(t *testing.T) {
	t.Run("dedupes circular required-with groups", func(t *testing.T) {
		// RequiredWith does NOT include the owning option, but A requires [B,C],
		// B requires [A,C], C requires [A,B] - all produce sorted key "a,b,c".
		options := map[string]*Option{
			"a": {Name: "a", RequiredWith: []string{"b", "c"}},
			"b": {Name: "b", RequiredWith: []string{"a", "c"}},
			"c": {Name: "c", RequiredWith: []string{"a", "b"}},
		}

		got := translateMmv1RequiredWith(options)
		want := [][]string{{"a", "b", "c"}}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("getRequiredTogether() = %#v, want %#v", got, want)
		}
	})

	t.Run("returns nil when no required-with", func(t *testing.T) {
		options := map[string]*Option{
			"a": {Name: "a"},
			"b": {Name: "b"},
		}

		if got := translateMmv1RequiredWith(options); got != nil {
			t.Fatalf("getRequiredTogether() = %#v, want nil", got)
		}
	})

	t.Run("normalizes dotted required-with names", func(t *testing.T) {
		options := map[string]*Option{
			"fooBar": {Name: "fooBar", RequiredWith: []string{"parent.bar"}},
		}

		got := translateMmv1RequiredWith(options)
		want := [][]string{{"bar", "foo_bar"}}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("getRequiredTogether() = %#v, want %#v", got, want)
		}
	})

	t.Run("drops group when only TF-only peers remain", func(t *testing.T) {
		options := map[string]*Option{
			"foo": {Name: "foo", RequiredWith: []string{"effectiveLabels", "terraformLabels"}},
		}

		if got := translateMmv1RequiredWith(options); got != nil {
			t.Fatalf("getRequiredTogether() = %#v, want nil", got)
		}
	})

	t.Run("keeps group when TF-only peers are stripped but normal ones remain", func(t *testing.T) {
		options := map[string]*Option{
			"foo": {Name: "foo", RequiredWith: []string{"effectiveLabels", "bar"}},
		}

		got := translateMmv1RequiredWith(options)
		want := [][]string{{"bar", "foo"}}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("getRequiredTogether() = %#v, want %#v", got, want)
		}
	})
}

func TestMapMmv1ToAnsible(t *testing.T) {
	cases := []struct {
		mmv1Type string
		want     Type
	}{
		{"String", TypeStr},
		{"Integer", TypeInt},
		{"Boolean", TypeBool},
		{"NestedObject", TypeDict},
		{"KeyValueAnnotations", TypeDict},
		{"KeyValueLabels", TypeDict},
		{"KeyValuePairs", TypeDict},
		{"KeyValueEffectiveLabels", TypeDict},
		{"KeyValueTerraformLabels", TypeDict},
		{"Array", TypeList},
		{"Enum", TypeStr},
		{"ResourceRef", TypeDict},
		{"Fingerprint", TypeStr},
		{"Time", TypeStr},
	}

	for _, tc := range cases {
		t.Run(tc.mmv1Type+"_maps_to_"+string(tc.want), func(t *testing.T) {
			prop := &mmv1api.Type{Type: tc.mmv1Type}
			got := MapMmv1ToAnsible(prop)
			if got != tc.want {
				t.Fatalf("MapMmv1ToAnsible(%q) = %q, want %q", tc.mmv1Type, got, tc.want)
			}
		})
	}

	t.Run("nil property returns empty type", func(t *testing.T) {
		if got := MapMmv1ToAnsible(nil); got != "" {
			t.Fatalf("MapMmv1ToAnsible(nil) = %q, want empty string", got)
		}
	})

	t.Run("unknown type falls back to str", func(t *testing.T) {
		prop := &mmv1api.Type{Type: "SomeUnknownType"}
		got := MapMmv1ToAnsible(prop)
		if got != TypeStr {
			t.Fatalf("MapMmv1ToAnsible(unknown) = %q, want %q", got, TypeStr)
		}
	})
}

func TestLooksLikeSensitiveField(t *testing.T) {
	t.Run("returns false for empty name", func(t *testing.T) {
		if looksLikeSensitiveField("") {
			t.Fatal("looksLikeSensitiveField(\"\") = true, want false")
		}
	})

	sensitive := []string{
		"password", "userPassword", "myPasswd", "secretValue",
		"accessToken", "apiKey", "api_key", "privateKey", "private_key",
		"serviceCredential", "authHeader", "authorization",
		"tlsCertificate", "clientCert",
	}
	for _, name := range sensitive {
		name := name
		t.Run(name+"_is_sensitive", func(t *testing.T) {
			if !looksLikeSensitiveField(name) {
				t.Fatalf("looksLikeSensitiveField(%q) = false, want true", name)
			}
		})
	}

	plain := []string{"displayName", "region", "project", "location", "description"}
	for _, name := range plain {
		name := name
		t.Run(name+"_is_not_sensitive", func(t *testing.T) {
			if looksLikeSensitiveField(name) {
				t.Fatalf("looksLikeSensitiveField(%q) = true, want false", name)
			}
		})
	}
}

func TestOptionOutputOnly(t *testing.T) {
	t.Run("returns true when description contains 'output only'", func(t *testing.T) {
		o := &Option{Description: []string{"This field is output only."}}
		if !o.OutputOnly() {
			t.Fatal("OutputOnly() = false, want true")
		}
	})

	t.Run("returns true for uppercase OUTPUT ONLY", func(t *testing.T) {
		o := &Option{Description: []string{"OUTPUT ONLY field."}}
		if !o.OutputOnly() {
			t.Fatal("OutputOnly() = false, want true for uppercase")
		}
	})

	t.Run("returns true when phrase spans multiple description strings", func(t *testing.T) {
		o := &Option{Description: []string{"This is an", "output only value."}}
		if !o.OutputOnly() {
			t.Fatal("OutputOnly() = false, want true across joined strings")
		}
	})

	t.Run("returns false when description does not mention output only", func(t *testing.T) {
		o := &Option{Description: []string{"The name of the resource."}}
		if o.OutputOnly() {
			t.Fatal("OutputOnly() = true, want false")
		}
	})

	t.Run("returns false for empty description", func(t *testing.T) {
		o := &Option{}
		if o.OutputOnly() {
			t.Fatal("OutputOnly() = true, want false for empty description")
		}
	})
}

func TestOptionAnsibleName(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"fooBar", "foo_bar"},
		{"displayName", "display_name"},
		{"project", "project"},
		{"myLongFieldName", "my_long_field_name"},
		{"alreadySnakeCase", "already_snake_case"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name+"_becomes_"+tc.want, func(t *testing.T) {
			o := &Option{Name: tc.name}
			got := o.AnsibleName()
			if got != tc.want {
				t.Fatalf("AnsibleName() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestOptionApiName(t *testing.T) {
	t.Run("returns Mmv1.ApiName when set", func(t *testing.T) {
		o := &Option{Name: "myField", Mmv1: &mmv1api.Type{ApiName: "myApiField"}}
		if got := o.ApiName(); got != "myApiField" {
			t.Fatalf("ApiName() = %q, want %q", got, "myApiField")
		}
	})

	t.Run("falls back to Name when Mmv1.ApiName is empty", func(t *testing.T) {
		o := &Option{Name: "myField", Mmv1: &mmv1api.Type{ApiName: ""}}
		if got := o.ApiName(); got != "myField" {
			t.Fatalf("ApiName() = %q, want %q", got, "myField")
		}
	})

	t.Run("falls back to Name when Mmv1 is nil", func(t *testing.T) {
		o := &Option{Name: "myField", Mmv1: nil}
		if got := o.ApiName(); got != "myField" {
			t.Fatalf("ApiName() = %q, want %q", got, "myField")
		}
	})
}

func TestOptionIsList(t *testing.T) {
	t.Run("returns true for TypeList", func(t *testing.T) {
		o := &Option{Type: TypeList}
		if !o.IsList() {
			t.Fatal("IsList() = false, want true")
		}
	})

	t.Run("returns false for TypeDict", func(t *testing.T) {
		o := &Option{Type: TypeDict}
		if o.IsList() {
			t.Fatal("IsList() = true, want false")
		}
	})

	t.Run("returns false for TypeStr", func(t *testing.T) {
		o := &Option{Type: TypeStr}
		if o.IsList() {
			t.Fatal("IsList() = true, want false")
		}
	})
}

func TestOptionIsNestedObject(t *testing.T) {
	t.Run("returns true when Mmv1.Type is NestedObject", func(t *testing.T) {
		o := &Option{Mmv1: &mmv1api.Type{Type: "NestedObject"}}
		if !o.IsNestedObject() {
			t.Fatal("IsNestedObject() = false, want true")
		}
	})

	t.Run("returns false when Mmv1.Type is String", func(t *testing.T) {
		o := &Option{Mmv1: &mmv1api.Type{Type: "String"}}
		if o.IsNestedObject() {
			t.Fatal("IsNestedObject() = true, want false")
		}
	})
}

func TestOptionIsNestedList(t *testing.T) {
	t.Run("returns true when TypeList with NestedObject elements", func(t *testing.T) {
		o := &Option{
			Type: TypeList,
			Mmv1: &mmv1api.Type{
				Type:     "Array",
				ItemType: &mmv1api.Type{Type: "NestedObject"},
			},
		}
		if !o.IsNestedList() {
			t.Fatal("IsNestedList() = false, want true")
		}
	})

	t.Run("returns false when TypeList with String elements", func(t *testing.T) {
		o := &Option{
			Type: TypeList,
			Mmv1: &mmv1api.Type{
				Type:     "Array",
				ItemType: &mmv1api.Type{Type: "String"},
			},
		}
		if o.IsNestedList() {
			t.Fatal("IsNestedList() = true, want false for non-nested list")
		}
	})

	t.Run("returns false when not a list", func(t *testing.T) {
		o := &Option{
			Type: TypeDict,
			Mmv1: &mmv1api.Type{Type: "NestedObject"},
		}
		if o.IsNestedList() {
			t.Fatal("IsNestedList() = true, want false for non-list type")
		}
	})
}

func TestOptionClassName(t *testing.T) {
	t.Run("root-level nested object returns camelized name", func(t *testing.T) {
		o := &Option{
			Name:   "networkConfig",
			Parent: nil,
			Mmv1:   &mmv1api.Type{Type: "NestedObject"},
		}
		got := o.ClassName()
		if got != "NetworkConfig" {
			t.Fatalf("ClassName() = %q, want %q", got, "NetworkConfig")
		}
	})

	t.Run("child nested object prepends parent class name", func(t *testing.T) {
		parent := &Option{
			Name: "networkConfig",
			Mmv1: &mmv1api.Type{Type: "NestedObject"},
		}
		child := &Option{
			Name:   "subnetConfig",
			Parent: parent,
			Mmv1:   &mmv1api.Type{Type: "NestedObject"},
		}
		got := child.ClassName()
		if got != "NetworkConfigSubnetConfig" {
			t.Fatalf("ClassName() = %q, want %q", got, "NetworkConfigSubnetConfig")
		}
	})

	t.Run("nested list strips trailing s from name and prepends parent", func(t *testing.T) {
		parent := &Option{
			Name: "cluster",
			Mmv1: &mmv1api.Type{Type: "NestedObject"},
		}
		child := &Option{
			Name:   "nodes",
			Parent: parent,
			Type:   TypeList,
			Mmv1: &mmv1api.Type{
				Type:     "Array",
				ItemType: &mmv1api.Type{Type: "NestedObject"},
			},
		}
		got := child.ClassName()
		if got != "ClusterNode" {
			t.Fatalf("ClassName() = %q, want %q", got, "ClusterNode")
		}
	})

	t.Run("root-level nested list returns camelized singular name", func(t *testing.T) {
		o := &Option{
			Name:   "items",
			Parent: nil,
			Type:   TypeList,
			Mmv1: &mmv1api.Type{
				Type:     "Array",
				ItemType: &mmv1api.Type{Type: "NestedObject"},
			},
		}
		// No parent → falls through to the plain Camelize branch
		got := o.ClassName()
		if got != "Items" {
			t.Fatalf("ClassName() = %q, want %q", got, "Items")
		}
	})
}

func TestOptionSuboptions(t *testing.T) {
	outputOpt := &Option{Name: "status", Output: true}
	virtualOpt := &Option{Name: "virt", Virtual: true}
	clientOpt := &Option{Name: "client", ClientSide: true}
	urlOpt := &Option{Name: "urlOnly", Mmv1: &mmv1api.Type{UrlParamOnly: true}}
	plainOpt := &Option{Name: "region"}
	outputOnlyDescOpt := &Option{Name: "selfLink", Output: false, Description: []string{"output only link"}}

	parent := &Option{
		Name: "config",
		Suboptions: map[string]*Option{
			"status":   outputOpt,
			"virt":     virtualOpt,
			"client":   clientOpt,
			"urlOnly":  urlOpt,
			"region":   plainOpt,
			"selfLink": outputOnlyDescOpt,
		},
	}

	t.Run("OutputSuboptions returns only Output=true options", func(t *testing.T) {
		got := parent.OutputSuboptions()
		if _, ok := got["status"]; !ok {
			t.Fatal("OutputSuboptions() missing 'status'")
		}
		if len(got) != 1 {
			t.Fatalf("OutputSuboptions() = %d entries, want 1", len(got))
		}
	})

	t.Run("InputSuboptions excludes Output, Virtual, ClientSide, UrlParamOnly", func(t *testing.T) {
		got := parent.InputSuboptions()
		for _, excluded := range []string{"status", "virt", "client", "urlOnly"} {
			if _, ok := got[excluded]; ok {
				t.Fatalf("InputSuboptions() contains %q, should be excluded", excluded)
			}
		}
		if _, ok := got["region"]; !ok {
			t.Fatal("InputSuboptions() missing 'region'")
		}
	})

	t.Run("ArgumentSuboptions excludes Output=true and OutputOnly() description options", func(t *testing.T) {
		got := parent.ArgumentSuboptions()
		if _, ok := got["status"]; ok {
			t.Fatal("ArgumentSuboptions() contains 'status' (Output=true), should be excluded")
		}
		if _, ok := got["selfLink"]; ok {
			t.Fatal("ArgumentSuboptions() contains 'selfLink' (OutputOnly desc), should be excluded")
		}
		if _, ok := got["region"]; !ok {
			t.Fatal("ArgumentSuboptions() missing 'region'")
		}
	})
}
