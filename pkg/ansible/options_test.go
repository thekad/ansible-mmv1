// Copyright 2025 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0

package ansible

import (
	"reflect"
	"testing"
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

		// after filtering effectiveLabels, only "foo" survives — group < 2, must be nil
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
		// B requires [A,C], C requires [A,B] — all produce sorted key "a,b,c".
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
