// Copyright 2025 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0

package renderer

import (
	"reflect"
	"testing"
)

// ---------------------------------------------------------------------------
// splitLinesFunc
// ---------------------------------------------------------------------------

func TestSplitLinesFunc(t *testing.T) {
	t.Run("splits on newline", func(t *testing.T) {
		got := splitLinesFunc("foo\nbar\nbaz")
		want := []string{"foo", "bar", "baz"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("splitLinesFunc() = %#v, want %#v", got, want)
		}
	})

	t.Run("trims leading and trailing whitespace per line", func(t *testing.T) {
		got := splitLinesFunc("  foo  \n  bar  ")
		want := []string{"foo", "bar"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("splitLinesFunc() = %#v, want %#v", got, want)
		}
	})

	t.Run("drops blank lines", func(t *testing.T) {
		got := splitLinesFunc("foo\n\n\nbar")
		want := []string{"foo", "bar"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("splitLinesFunc() = %#v, want %#v", got, want)
		}
	})

	t.Run("drops whitespace-only lines", func(t *testing.T) {
		got := splitLinesFunc("foo\n   \nbar")
		want := []string{"foo", "bar"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("splitLinesFunc() = %#v, want %#v", got, want)
		}
	})

	t.Run("empty string returns empty slice", func(t *testing.T) {
		got := splitLinesFunc("")
		if len(got) != 0 {
			t.Fatalf("splitLinesFunc(\"\") = %#v, want []", got)
		}
	})

	t.Run("single line with no newline", func(t *testing.T) {
		got := splitLinesFunc("hello")
		want := []string{"hello"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("splitLinesFunc() = %#v, want %#v", got, want)
		}
	})
}

// ---------------------------------------------------------------------------
// indentFunc
// ---------------------------------------------------------------------------

func TestIndentFunc(t *testing.T) {
	t.Run("indents all lines when first=true", func(t *testing.T) {
		got := indentFunc(4, true, "foo\nbar")
		want := "    foo\n    bar"
		if got != want {
			t.Fatalf("indentFunc() = %q, want %q", got, want)
		}
	})

	t.Run("skips indenting the first line when first=false", func(t *testing.T) {
		got := indentFunc(4, false, "foo\nbar\nbaz")
		want := "foo\n    bar\n    baz"
		if got != want {
			t.Fatalf("indentFunc() = %q, want %q", got, want)
		}
	})

	t.Run("zero spaces produces no indentation", func(t *testing.T) {
		got := indentFunc(0, true, "foo\nbar")
		want := "foo\nbar"
		if got != want {
			t.Fatalf("indentFunc(0) = %q, want %q", got, want)
		}
	})

	t.Run("negative spaces treated as zero", func(t *testing.T) {
		got := indentFunc(-5, true, "foo")
		want := "foo"
		if got != want {
			t.Fatalf("indentFunc(-5) = %q, want %q", got, want)
		}
	})

	t.Run("empty text returned unchanged", func(t *testing.T) {
		got := indentFunc(4, true, "")
		if got != "" {
			t.Fatalf("indentFunc(empty) = %q, want \"\"", got)
		}
	})

	t.Run("single line indented correctly", func(t *testing.T) {
		got := indentFunc(2, true, "hello")
		want := "  hello"
		if got != want {
			t.Fatalf("indentFunc() = %q, want %q", got, want)
		}
	})
}

// ---------------------------------------------------------------------------
// singularFunc
// ---------------------------------------------------------------------------

func TestSingularFunc(t *testing.T) {
	t.Run("strips trailing s", func(t *testing.T) {
		if got := singularFunc("nodes"); got != "node" {
			t.Fatalf("singularFunc(\"nodes\") = %q, want \"node\"", got)
		}
	})

	t.Run("leaves non-s-ending words unchanged", func(t *testing.T) {
		if got := singularFunc("config"); got != "config" {
			t.Fatalf("singularFunc(\"config\") = %q, want \"config\"", got)
		}
	})

	t.Run("empty string returns empty string", func(t *testing.T) {
		if got := singularFunc(""); got != "" {
			t.Fatalf("singularFunc(\"\") = %q, want \"\"", got)
		}
	})

	t.Run("single character s returns empty string", func(t *testing.T) {
		if got := singularFunc("s"); got != "" {
			t.Fatalf("singularFunc(\"s\") = %q, want \"\"", got)
		}
	})

	t.Run("word ending in ss strips only one s", func(t *testing.T) {
		if got := singularFunc("lass"); got != "las" {
			t.Fatalf("singularFunc(\"lass\") = %q, want \"las\"", got)
		}
	})
}

// ---------------------------------------------------------------------------
// sortedKeysFunc
// ---------------------------------------------------------------------------

func TestSortedKeysFunc(t *testing.T) {
	t.Run("returns keys in alphabetical order", func(t *testing.T) {
		m := map[string]int{"banana": 2, "apple": 1, "cherry": 3}
		got := sortedKeysFunc(m)
		want := []string{"apple", "banana", "cherry"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("sortedKeysFunc() = %#v, want %#v", got, want)
		}
	})

	t.Run("works with map[string]string", func(t *testing.T) {
		m := map[string]string{"z": "last", "a": "first", "m": "middle"}
		got := sortedKeysFunc(m)
		want := []string{"a", "m", "z"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("sortedKeysFunc() = %#v, want %#v", got, want)
		}
	})

	t.Run("empty map returns empty slice", func(t *testing.T) {
		m := map[string]int{}
		got := sortedKeysFunc(m)
		if len(got) != 0 {
			t.Fatalf("sortedKeysFunc(empty) = %#v, want []", got)
		}
	})

	t.Run("nil input returns nil", func(t *testing.T) {
		if got := sortedKeysFunc(nil); got != nil {
			t.Fatalf("sortedKeysFunc(nil) = %#v, want nil", got)
		}
	})

	t.Run("non-map input returns nil", func(t *testing.T) {
		if got := sortedKeysFunc("not a map"); got != nil {
			t.Fatalf("sortedKeysFunc(string) = %#v, want nil", got)
		}
	})

	t.Run("single-entry map returns single key", func(t *testing.T) {
		m := map[string]bool{"only": true}
		got := sortedKeysFunc(m)
		want := []string{"only"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("sortedKeysFunc() = %#v, want %#v", got, want)
		}
	})
}

// ---------------------------------------------------------------------------
// lenFunc
// ---------------------------------------------------------------------------

func TestLenFunc(t *testing.T) {
	t.Run("nil returns 0", func(t *testing.T) {
		if got := lenFunc(nil); got != 0 {
			t.Fatalf("lenFunc(nil) = %d, want 0", got)
		}
	})

	t.Run("string length", func(t *testing.T) {
		if got := lenFunc("hello"); got != 5 {
			t.Fatalf("lenFunc(\"hello\") = %d, want 5", got)
		}
	})

	t.Run("slice length", func(t *testing.T) {
		if got := lenFunc([]int{1, 2, 3}); got != 3 {
			t.Fatalf("lenFunc([]int{1,2,3}) = %d, want 3", got)
		}
	})

	t.Run("map length", func(t *testing.T) {
		if got := lenFunc(map[string]int{"a": 1, "b": 2}); got != 2 {
			t.Fatalf("lenFunc(map) = %d, want 2", got)
		}
	})

	t.Run("empty slice returns 0", func(t *testing.T) {
		if got := lenFunc([]string{}); got != 0 {
			t.Fatalf("lenFunc(empty slice) = %d, want 0", got)
		}
	})

	t.Run("non-collection type returns 0", func(t *testing.T) {
		if got := lenFunc(42); got != 0 {
			t.Fatalf("lenFunc(int) = %d, want 0", got)
		}
	})
}

// ---------------------------------------------------------------------------
// compareValues / gtFunc / ltFunc / eqFunc / neFunc / gteFunc / lteFunc
// ---------------------------------------------------------------------------

func TestCompareValues(t *testing.T) {
	t.Run("equal integers return 0", func(t *testing.T) {
		if got := compareValues(3, 3); got != 0 {
			t.Fatalf("compareValues(3, 3) = %d, want 0", got)
		}
	})

	t.Run("a < b returns negative", func(t *testing.T) {
		if got := compareValues(1, 5); got >= 0 {
			t.Fatalf("compareValues(1, 5) = %d, want < 0", got)
		}
	})

	t.Run("a > b returns positive", func(t *testing.T) {
		if got := compareValues(5, 1); got <= 0 {
			t.Fatalf("compareValues(5, 1) = %d, want > 0", got)
		}
	})

	t.Run("float values are compared as truncated int64", func(t *testing.T) {
		// both 2.9 and 2.1 truncate to 2, so equal
		if got := compareValues(2.9, 2.1); got != 0 {
			t.Fatalf("compareValues(2.9, 2.1) = %d, want 0 (truncated)", got)
		}
	})

	t.Run("uint values compare correctly", func(t *testing.T) {
		if got := compareValues(uint(10), uint(20)); got >= 0 {
			t.Fatalf("compareValues(uint10, uint20) = %d, want < 0", got)
		}
	})
}

// These comparison functions are designed for use in Go template pipelines where the
// piped value is passed as the second argument (b). For example:
//
//	{{ .Count | gt 0 }}  →  gtFunc(0, count)  →  "is count > 0?"
//
// So gtFunc(a, b) answers "is b > a?", ltFunc(a, b) answers "is b < a?", etc.
// eqFunc and neFunc are symmetric so argument order does not matter.
func TestComparisonFuncs(t *testing.T) {
	// gtFunc(a, b) → "is b > a?"
	t.Run("gtFunc: b=5 > a=3 is true", func(t *testing.T) {
		if !gtFunc(3, 5) {
			t.Fatal("gtFunc(3, 5) = false, want true (b=5 > a=3)")
		}
	})
	t.Run("gtFunc: b=3 > a=5 is false", func(t *testing.T) {
		if gtFunc(5, 3) {
			t.Fatal("gtFunc(5, 3) = true, want false (b=3 > a=5 is false)")
		}
	})
	t.Run("gtFunc: equal values is false", func(t *testing.T) {
		if gtFunc(4, 4) {
			t.Fatal("gtFunc(4, 4) = true, want false")
		}
	})

	// ltFunc(a, b) → "is b < a?"
	t.Run("ltFunc: b=3 < a=5 is true", func(t *testing.T) {
		if !ltFunc(5, 3) {
			t.Fatal("ltFunc(5, 3) = false, want true (b=3 < a=5)")
		}
	})
	t.Run("ltFunc: b=5 < a=3 is false", func(t *testing.T) {
		if ltFunc(3, 5) {
			t.Fatal("ltFunc(3, 5) = true, want false (b=5 < a=3 is false)")
		}
	})

	// eqFunc / neFunc are symmetric
	t.Run("eqFunc: 4 == 4 is true", func(t *testing.T) {
		if !eqFunc(4, 4) {
			t.Fatal("eqFunc(4, 4) = false, want true")
		}
	})
	t.Run("eqFunc: 4 == 5 is false", func(t *testing.T) {
		if eqFunc(4, 5) {
			t.Fatal("eqFunc(4, 5) = true, want false")
		}
	})
	t.Run("neFunc: 4 != 5 is true", func(t *testing.T) {
		if !neFunc(4, 5) {
			t.Fatal("neFunc(4, 5) = false, want true")
		}
	})
	t.Run("neFunc: 4 != 4 is false", func(t *testing.T) {
		if neFunc(4, 4) {
			t.Fatal("neFunc(4, 4) = true, want false")
		}
	})

	// gteFunc(a, b) → "is b >= a?"
	t.Run("gteFunc: b=5 >= a=5 is true", func(t *testing.T) {
		if !gteFunc(5, 5) {
			t.Fatal("gteFunc(5, 5) = false, want true")
		}
	})
	t.Run("gteFunc: b=5 >= a=4 is true", func(t *testing.T) {
		if !gteFunc(4, 5) {
			t.Fatal("gteFunc(4, 5) = false, want true (b=5 >= a=4)")
		}
	})
	t.Run("gteFunc: b=3 >= a=5 is false", func(t *testing.T) {
		if gteFunc(5, 3) {
			t.Fatal("gteFunc(5, 3) = true, want false (b=3 >= a=5 is false)")
		}
	})

	// lteFunc(a, b) → "is b <= a?"
	t.Run("lteFunc: b=4 <= a=4 is true", func(t *testing.T) {
		if !lteFunc(4, 4) {
			t.Fatal("lteFunc(4, 4) = false, want true")
		}
	})
	t.Run("lteFunc: b=3 <= a=4 is true", func(t *testing.T) {
		if !lteFunc(4, 3) {
			t.Fatal("lteFunc(4, 3) = false, want true (b=3 <= a=4)")
		}
	})
	t.Run("lteFunc: b=5 <= a=4 is false", func(t *testing.T) {
		if lteFunc(4, 5) {
			t.Fatal("lteFunc(4, 5) = true, want false (b=5 <= a=4 is false)")
		}
	})
}

// ---------------------------------------------------------------------------
// tojsonFunc
// ---------------------------------------------------------------------------

func TestTojsonFunc(t *testing.T) {
	t.Run("marshals string", func(t *testing.T) {
		if got := tojsonFunc("hello"); got != `"hello"` {
			t.Fatalf("tojsonFunc(\"hello\") = %q, want %q", got, `"hello"`)
		}
	})

	t.Run("marshals integer", func(t *testing.T) {
		if got := tojsonFunc(42); got != "42" {
			t.Fatalf("tojsonFunc(42) = %q, want \"42\"", got)
		}
	})

	t.Run("marshals boolean true", func(t *testing.T) {
		if got := tojsonFunc(true); got != "true" {
			t.Fatalf("tojsonFunc(true) = %q, want \"true\"", got)
		}
	})

	t.Run("marshals nil as null", func(t *testing.T) {
		if got := tojsonFunc(nil); got != "null" {
			t.Fatalf("tojsonFunc(nil) = %q, want \"null\"", got)
		}
	})

	t.Run("marshals slice", func(t *testing.T) {
		if got := tojsonFunc([]int{1, 2, 3}); got != "[1,2,3]" {
			t.Fatalf("tojsonFunc([]int) = %q, want \"[1,2,3]\"", got)
		}
	})

	t.Run("marshals map with sorted keys", func(t *testing.T) {
		// Use a struct for deterministic output.
		type pair struct {
			A string `json:"a"`
			B int    `json:"b"`
		}
		got := tojsonFunc(pair{"hello", 1})
		want := `{"a":"hello","b":1}`
		if got != want {
			t.Fatalf("tojsonFunc(struct) = %q, want %q", got, want)
		}
	})
}

// ---------------------------------------------------------------------------
// concatFunc
// ---------------------------------------------------------------------------

func TestConcatFunc(t *testing.T) {
	t.Run("concatenates multiple strings", func(t *testing.T) {
		if got := concatFunc("foo", "bar", "baz"); got != "foobarbaz" {
			t.Fatalf("concatFunc() = %q, want \"foobarbaz\"", got)
		}
	})

	t.Run("single argument returned as-is", func(t *testing.T) {
		if got := concatFunc("only"); got != "only" {
			t.Fatalf("concatFunc(\"only\") = %q, want \"only\"", got)
		}
	})

	t.Run("no arguments returns empty string", func(t *testing.T) {
		if got := concatFunc(); got != "" {
			t.Fatalf("concatFunc() = %q, want \"\"", got)
		}
	})

	t.Run("empty strings are concatenated to empty", func(t *testing.T) {
		if got := concatFunc("", "", ""); got != "" {
			t.Fatalf("concatFunc(\"\",\"\",\"\") = %q, want \"\"", got)
		}
	})
}

// ---------------------------------------------------------------------------
// isNilFunc
// ---------------------------------------------------------------------------

func TestIsNilFunc(t *testing.T) {
	t.Run("nil returns true", func(t *testing.T) {
		if !isNilFunc(nil) {
			t.Fatal("isNilFunc(nil) = false, want true")
		}
	})

	t.Run("non-nil string returns false", func(t *testing.T) {
		if isNilFunc("hello") {
			t.Fatal("isNilFunc(\"hello\") = true, want false")
		}
	})

	t.Run("non-nil integer returns false", func(t *testing.T) {
		if isNilFunc(0) {
			t.Fatal("isNilFunc(0) = true, want false")
		}
	})

	t.Run("non-nil pointer to struct returns false", func(t *testing.T) {
		v := struct{}{}
		if isNilFunc(&v) {
			t.Fatal("isNilFunc(&struct{}{}) = true, want false")
		}
	})
}
