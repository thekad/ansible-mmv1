// Copyright 2025 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0

package ansible

import (
	"sort"
	"strings"

	mmv1api "github.com/GoogleCloudPlatform/magic-modules/mmv1/api"
	"github.com/GoogleCloudPlatform/magic-modules/mmv1/google"
	"github.com/rs/zerolog/log"
)

// Type represents the data types supported by Ansible modules
type Type string

// Ansible module data types as defined in the official documentation
const (
	TypeStr     Type = "str"
	TypeInt     Type = "int"
	TypeBool    Type = "bool"
	TypeList    Type = "list"
	TypeDict    Type = "dict"
	TypePath    Type = "path"
	TypeRaw     Type = "raw"
	TypeJsonarg Type = "jsonarg"
	TypeBytes   Type = "bytes"
	TypeBits    Type = "bits"
	TypeFloat   Type = "float"
)

// String returns the string representation of the AnsibleType
func (t Type) String() string {
	return string(t)
}

// MapMmv1ToAnsible maps magic-modules API types to Ansible module types
// Returns AnsibleType enum and error for better error handling
func MapMmv1ToAnsible(property *mmv1api.Type) Type {
	if property == nil {
		return ""
	}

	switch property.Type {
	case "String":
		return TypeStr
	case "Integer":
		return TypeInt
	case "Boolean":
		return TypeBool
	case "NestedObject":
		return TypeDict
	case "KeyValueAnnotations":
		return TypeDict
	case "KeyValueLabels":
		return TypeDict
	case "KeyValuePairs":
		return TypeDict
	case "KeyValueEffectiveLabels":
		return TypeDict
	case "KeyValueTerraformLabels":
		return TypeDict
	case "Array":
		return TypeList
	case "Enum":
		return TypeStr
	case "ResourceRef":
		return TypeDict
	case "Fingerprint":
		return TypeStr
	case "Time":
		return TypeStr
	default:
		log.Warn().Msgf("unknown API type '%s' defaulting to string", property.Type)
		return TypeStr
	}
}

type Dependency struct {
	// MutuallyExclusive is optional - list of options that cannot be used together
	MutuallyExclusive [][]string `yaml:"mutually_exclusive,omitempty"`

	// RequiredTogether is optional - list of options that must be used together
	RequiredTogether [][]string `yaml:"required_together,omitempty"`

	// RequiredOneOf is optional - list of option groups where at least one option must be provided
	RequiredOneOf [][]string `yaml:"required_one_of,omitempty"`
}

// Option represents a single option in the Ansible module documentation
// Based on: https://docs.ansible.com/ansible/latest/dev_guide/developing_modules_documenting.html#documentation-block
type Option struct {
	// Name is the name of the option
	Name string `yaml:"-"`

	// Parent is a reference to the parent option
	Parent *Option `yaml:"-"`

	// Mmv1 is a reference to the original MMv1 property
	Mmv1 *mmv1api.Type `yaml:"-"`

	// Description is required - explanation of what this option does
	// Can be a string or list of strings (each string is one paragraph)
	Description []string `yaml:"description"`

	// Type is optional - data type of the option
	// Uses AnsibleType enum for type safety
	Type Type `yaml:"type,omitempty"`

	// Default is optional - default value for the option
	Default interface{} `yaml:"default,omitempty"`

	// Required is optional - whether this option is required
	// Defaults to false if not specified
	Required bool `yaml:"required,omitempty"`

	// Choices is optional - list of valid values for this option
	Choices []string `yaml:"choices,omitempty"`

	// Elements is optional - if type='list', specifies the data type of list elements
	Elements Type `yaml:"elements,omitempty"`

	// Suboptions is optional - for complex types (dict), defines nested options
	Suboptions map[string]*Option `yaml:"suboptions,omitempty"`

	// Conflicts is optional - list of options that cannot be used together
	Conflicts []string `yaml:"-"`

	// RequiredWith is optional - list of options that must be used together with this option
	RequiredWith []string `yaml:"-"`

	// ExactlyOneOf is optional - list of options where exactly one must be provided
	ExactlyOneOf []string `yaml:"-"`

	// AtLeastOneOf is optional - list of options where at least one must be provided
	AtLeastOneOf []string `yaml:"-"`

	// NoLog is optional - whether this option is sensitive and should not be logged
	// Uses *bool to support three states: nil (absent), false (explicitly not sensitive), true (explicitly sensitive)
	NoLog *bool `yaml:"-"`

	// Output is optional - whether this option is output-only
	Output bool `yaml:"-"`

	// ClientSide is optional - whether this option is client-only
	ClientSide bool `yaml:"-"`

	// Virtual is optional - whether this option is virtual
	Virtual bool `yaml:"-"`

	// Dependency is optional - dependency constraints for this option
	Dependency *Dependency `yaml:"-"`

	// SendEmptyValue is optional - if true, we will include the empty value in requests to the API
	SendEmptyValue bool `yaml:"-"`

	// AllowEmptyObject is optional - if true, empty objects are sent to / read from the API instead of ignoring them
	AllowEmptyObject bool `yaml:"-"`
}

// HasNoLog returns true if NoLog is explicitly set (either true or false)
func (o *Option) HasNoLog() bool {
	return o.NoLog != nil
}

// IsNoLog returns true if NoLog is explicitly set to true
func (o *Option) IsNoLog() bool {
	return o.NoLog != nil && *o.NoLog
}

// OutputOnly returns true if the description mentions "output only". Kinda lame :(
func (o *Option) OutputOnly() bool {
	if strings.Contains(strings.ToLower(strings.Join(o.Description, " ")), "output only") {
		return true
	}

	return false
}

func (o *Option) UrlParamOnly() bool {
	if o.Mmv1 == nil {
		return false
	}
	return o.Mmv1.UrlParamOnly
}

func (o *Option) OutputSuboptions() map[string]*Option {
	outputSuboptions := map[string]*Option{}
	for name, option := range o.Suboptions {
		if option.Output {
			outputSuboptions[name] = option
		}
	}
	return outputSuboptions
}

func (o *Option) InputSuboptions() map[string]*Option {
	inputSuboptions := map[string]*Option{}
	for name, option := range o.Suboptions {
		if option.Output || option.Virtual || option.ClientSide || option.UrlParamOnly() {
			continue
		}
		inputSuboptions[name] = option
	}
	return inputSuboptions
}

func (o *Option) ArgumentSuboptions() map[string]*Option {
	argumentSuboptions := map[string]*Option{}
	for name, option := range o.Suboptions {
		if option.Output || option.OutputOnly() {
			continue
		}
		argumentSuboptions[name] = option
	}
	return argumentSuboptions
}

func (o *Option) IsList() bool {
	return o.Type == TypeList
}

func (o *Option) IsNestedObject() bool {
	return o.Mmv1.IsA("NestedObject")
}

func (o *Option) IsNestedList() bool {
	return o.IsList() && o.ElementsAre("NestedObject")
}

func (o *Option) AnsibleName() string {
	return google.Underscore(o.Name)
}

func (o *Option) ClassName() string {
	if o.IsNestedList() {
		if o.Parent != nil {
			return o.Parent.ClassName() + google.Camelize(strings.TrimSuffix(o.Name, "s"), "upper")
		}
	}
	if o.IsNestedObject() {
		if o.Parent != nil {
			return o.Parent.ClassName() + google.Camelize(o.Name, "upper")
		}
	}

	return google.Camelize(o.Name, "upper")
}

func (o *Option) ElementsAre(q string) bool {
	return o.Mmv1.ItemType.IsA(q)
}

func (o *Option) ApiName() string {
	if o.Mmv1 != nil {
		if o.Mmv1.ApiName != "" {
			return o.Mmv1.ApiName
		}
	}
	return o.Name
}

// NewOptionsFromMmv1 creates a map of Ansible options from a magic-modules API Resource
// This constructor extracts user properties from the API Resource and converts them
// to Ansible module options following the documentation format
func NewOptionsFromMmv1(resource *mmv1api.Resource) map[string]*Option {
	if resource == nil {
		return nil
	}

	options := convertPropertiesToOptions(resource.AllUserProperties(), nil, false)
	virtualOptions := convertPropertiesToOptions(resource.UserVirtualFields(), nil, true)
	for name, option := range virtualOptions {
		options[name] = option
	}

	return options
}

// looksLikeSensitiveField checks if a property name looks like it should contain sensitive data
// based on common naming patterns for secrets, passwords, keys, tokens, etc.
func looksLikeSensitiveField(name string) bool {
	if name == "" {
		return false
	}

	// Convert to lowercase for case-insensitive matching
	lowerName := strings.ToLower(name)

	for _, pattern := range sensitiveFieldPatterns {
		if strings.Contains(lowerName, pattern) {
			return true
		}
	}

	return false
}

// convertPropertiesToOptions converts MMv1 properties to Ansible options
func convertPropertiesToOptions(properties []*mmv1api.Type, parent *Option, virtual bool) map[string]*Option {
	if properties == nil {
		return nil
	}

	options := map[string]*Option{}

	for _, property := range properties {

		if isTFOnlyPropertyName(property.Name) {
			continue
		}

		// Determine NoLog value with heuristics
		var noLog *bool
		falseVal := false
		trueVal := true
		if !property.Sensitive && looksLikeSensitiveField(property.Name) {
			// Explicitly set to false if it looks sensitive but isn't marked as such
			noLog = &falseVal
			log.Info().Str("property", property.Name).Msgf("property looks sensitive, explicitly setting NoLog to false")
		} else if property.Sensitive {
			noLog = &trueVal
		} else {
			noLog = nil
		}

		// Create the option
		option := &Option{
			Name:             property.Name,
			Mmv1:             property,
			Parent:           parent,
			Description:      parsePropertyDescription(property),
			Type:             MapMmv1ToAnsible(property),
			Required:         property.Required,
			Default:          property.DefaultValue,
			Choices:          property.EnumValues,
			Conflicts:        property.Conflicts,
			RequiredWith:     property.RequiredWith,
			ExactlyOneOf:     property.ExactlyOneOf,
			AtLeastOneOf:     property.AtLeastOneOf,
			NoLog:            noLog,
			Output:           property.Output,
			ClientSide:       property.ClientSide,
			Virtual:          virtual,
			SendEmptyValue:   property.SendEmptyValue,
			AllowEmptyObject: property.AllowEmptyObject,
		}

		// Handle list element types
		if option.Type == TypeList && property.ItemType != nil {
			option.Elements = MapMmv1ToAnsible(property.ItemType)

			// If the list contains nested objects, create suboptions for the element type
			if property.ItemType.Type == "NestedObject" && property.ItemType.Properties != nil {
				option.Suboptions = convertPropertiesToOptions(property.ItemType.Properties, option, false) // virtual fields are top level only (so far)
			}
		}

		// Handle nested dictionary objects (direct suboptions)
		if option.Type == TypeDict && property.Properties != nil {
			subOpts := convertPropertiesToOptions(property.Properties, option, false) // virtual fields are top level only (so far)
			option.Suboptions = subOpts
			option.Dependency = &Dependency{
				MutuallyExclusive: translateMmv1Conflicts(subOpts),
				RequiredTogether:  translateMmv1RequiredWith(subOpts),
				RequiredOneOf:     translateMmv1AtLeastOneOf(subOpts),
			}
		}

		options[option.AnsibleName()] = option
	}

	return options
}

// extractGroupsFromField iterates options and builds deduplicated sorted groups
// from the provided per-option field accessor.
// If prependOwner is true the owning option's AnsibleName is prepended to the group
// (for Conflicts / RequiredWith which do not include the owner in their lists).
// If false, the field's entries are assumed to already contain the owner
// (for AtLeastOneOf).
func extractGroupsFromField(
	options map[string]*Option,
	field func(*Option) []string,
	prependOwner bool,
) [][]string {
	if options == nil {
		return nil
	}

	var groups [][]string
	seen := map[string]bool{}

	for name, option := range options {
		if option == nil {
			continue
		}
		if isTFOnlyPropertyName(name) {
			continue
		}

		entries := field(option)
		if len(entries) == 0 {
			continue
		}

		normalized := []string{}
		for _, entry := range entries {
			parts := strings.Split(entry, ".")
			entryName := parts[len(parts)-1]
			if isTFOnlyPropertyName(entryName) {
				continue
			}
			normalized = append(normalized, google.Underscore(entryName))
		}

		group := []string{}
		if prependOwner {
			group = append(group, option.AnsibleName())
		}
		group = append(group, normalized...)

		if len(group) < 2 {
			continue
		}

		sort.Strings(group)
		key := strings.Join(group, ",")
		if !seen[key] {
			seen[key] = true
			groups = append(groups, group)
		}
	}

	if len(groups) == 0 {
		return nil
	}

	sortStringSlices(groups)
	return groups
}

// translateMmv1Conflicts analyzes Conflicts on each option and returns deduplicated
// mutually_exclusive groups.
func translateMmv1Conflicts(options map[string]*Option) [][]string {
	return extractGroupsFromField(options, func(o *Option) []string { return o.Conflicts }, true)
}

// translateMmv1RequiredWith analyzes RequiredWith on each option and returns deduplicated
// required_together groups.
func translateMmv1RequiredWith(options map[string]*Option) [][]string {
	return extractGroupsFromField(options, func(o *Option) []string { return o.RequiredWith }, true)
}

// translateMmv1AtLeastOneOf analyzes AtLeastOneOf on each option and returns deduplicated
// required_one_of groups.
func translateMmv1AtLeastOneOf(options map[string]*Option) [][]string {
	return extractGroupsFromField(options, func(o *Option) []string { return o.AtLeastOneOf }, false)
}

// sortStringSlices sorts a slice of string slices for stable, deterministic output.
// Sorting is done by comparing the joined strings of each inner slice.
func sortStringSlices(slices [][]string) {
	sort.Slice(slices, func(i, j int) bool {
		return strings.Join(slices[i], ",") < strings.Join(slices[j], ",")
	})
}
