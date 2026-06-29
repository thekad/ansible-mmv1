// Copyright 2025 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0

package ansible

import (
	"fmt"

	mmv1api "github.com/GoogleCloudPlatform/magic-modules/mmv1/api"
	"github.com/GoogleCloudPlatform/magic-modules/mmv1/google"
	"github.com/rs/zerolog/log"
	"github.com/thekad/ansible-mmv1/pkg/api"
)

// ReturnType represents the data types returned by the module
type ReturnType string

// ReturnType as defined in the official documentation
const (
	ReturnTypeStr     ReturnType = "str"
	ReturnTypeInt     ReturnType = "int"
	ReturnTypeBool    ReturnType = "bool"
	ReturnTypeList    ReturnType = "list"
	ReturnTypeDict    ReturnType = "dict"
	ReturnTypeFloat   ReturnType = "float"
	ReturnTypeComplex ReturnType = "complex"
)

// String returns the string representation of the AnsibleType
func (t ReturnType) ToString() string {
	return string(t)
}

// ReturnAttribute represents the returns section of the Ansible module documentation
// Based on: https://docs.ansible.com/ansible/latest/dev_guide/developing_modules_documenting.html#return-block
type ReturnAttribute struct {
	// Description - detailed description of what this value represents
	// Required field - string or list of strings, capitalized with trailing dot
	Description interface{} `yaml:"description"`

	// Returned - when this value is returned (e.g., "always", "changed", "success")
	// Required field - string with human-readable content
	Returned string `yaml:"returned"`

	// Type - data type of the returned value
	// Required field - one of the ReturnType constants
	Type ReturnType `yaml:"type"`

	// Elements - if type='list', specifies the data type of the list's elements
	// Optional field
	Elements ReturnType `yaml:"elements,omitempty"`

	// Contains - for nested return values (type: dict, list/elements: dict, or complex)
	// Optional field - map of nested ReturnAttribute objects
	Contains map[string]*ReturnAttribute `yaml:"contains,omitempty"`
}

type ReturnBlock struct {
	Returns map[string]*ReturnAttribute `yaml:"returns"`
}

func (rb *ReturnBlock) ToString() string {
	return ToYAML(rb.Returns)
}

// mapMmv1TypeToReturnType maps magic-modules API types to Ansible module return types
// Returns ReturnType enum and error for better error handling
func mapMmv1TypeToReturnType(property *mmv1api.Type) (ReturnType, error) {
	if property == nil {
		return "", fmt.Errorf("property is nil")
	}

	if property.Type == "" {
		return ReturnTypeStr, fmt.Errorf("property type is empty, defaulting to string")
	}

	switch property.Type {
	case "String":
		return ReturnTypeStr, nil
	case "Integer":
		return ReturnTypeInt, nil
	case "Boolean":
		return ReturnTypeBool, nil
	case "NestedObject":
		return ReturnTypeDict, nil
	case "KeyValueAnnotations":
		return ReturnTypeDict, nil
	case "KeyValueLabels":
		return ReturnTypeDict, nil
	case "KeyValuePairs":
		return ReturnTypeDict, nil
	case "Array":
		return ReturnTypeList, nil
	case "Enum":
		return ReturnTypeStr, nil
	case "Fingerprint":
		return ReturnTypeStr, nil
	case "Time":
		return ReturnTypeStr, nil
	default:
		return ReturnTypeStr, fmt.Errorf("unknown API type '%s' defaulting to string", property.Type)
	}
}

// changedReturnAttribute returns a ReturnAttribute for the standard "changed"
// key with the given description.
func changedReturnAttribute(description string) *ReturnAttribute {
	return &ReturnAttribute{
		Description: description,
		Returned:    "always",
		Type:        ReturnTypeBool,
	}
}

// NewReturnBlockFromMmv1 creates a map of Ansible return attributes from a magic-modules API Resource
// This function extracts properties from the API Resource and converts them to Ansible module return format
// following the specification at: https://docs.ansible.com/ansible/latest/dev_guide/developing_modules_documenting.html#return-block
func NewReturnBlockFromMmv1(resource *mmv1api.Resource) *ReturnBlock {
	if resource == nil {
		return &ReturnBlock{}
	}

	returns := &ReturnBlock{
		Returns: make(map[string]*ReturnAttribute),
	}

	// Add standard return values that all GCP modules should have
	returns.Returns["changed"] = changedReturnAttribute("Whether the resource was changed.")

	returns.Returns["state"] = &ReturnAttribute{
		Description: "The current state of the resource.",
		Returned:    "always",
		Type:        ReturnTypeStr,
	}

	// Process properties from the API Resource
	convertedReturns := convertPropertiesToReturns(google.Select(resource.GettableProperties(), func(p *mmv1api.Type) bool {
		return p.Output
	}), true)

	// Merge the converted returns with the standard returns
	for name, returnAttr := range convertedReturns {
		returns.Returns[name] = returnAttr
	}

	return returns
}

// convertPropertiesToReturns converts MMv1 properties to Ansible return attributes.
// When extended is true, additional annotations (ResourceRef links, immutable notes)
// are appended to property descriptions - suitable for regular modules.
// Set extended to false for info modules to emit only the core description sentences.
func convertPropertiesToReturns(properties []*mmv1api.Type, extended bool) map[string]*ReturnAttribute {
	if properties == nil {
		return nil
	}

	returns := make(map[string]*ReturnAttribute)

	for _, property := range properties {
		if property.Name == "effectiveLabels" || property.Name == "effectiveAnnotations" || property.Name == "terraformLabels" {
			log.Debug().Msgf("skipping TF-only property: %s", property.Name)
			continue
		}

		returnName := property.Name

		// Create the return attribute
		returnType, err := mapMmv1TypeToReturnType(property)
		if err != nil {
			log.Warn().Err(err).Msgf("error mapping return type for property %s", property.Name)
			continue
		}

		returnAttr := &ReturnAttribute{
			Description: parsePropertyDescription(property, extended),
			Returned:    determineReturnedCondition(property),
			Type:        returnType,
		}

		// Handle list element types
		if returnAttr.Type == ReturnTypeList && property.ItemType != nil {
			elementType, err := mapMmv1TypeToReturnType(property.ItemType)
			if err != nil {
				log.Warn().Err(err).Msgf("error mapping return element type for property %s", property.Name)
			}
			returnAttr.Elements = elementType

			// If the list contains nested objects, create contains for the element type
			if property.ItemType.Type == "NestedObject" && property.ItemType.Properties != nil {
				returnAttr.Contains = convertPropertiesToReturns(property.ItemType.Properties, extended)
			}
		}

		// Handle nested dictionary objects (direct contains)
		if (returnAttr.Type == ReturnTypeDict || returnAttr.Type == ReturnTypeComplex) && property.Properties != nil {
			returnAttr.Contains = convertPropertiesToReturns(property.Properties, extended)
		}

		returns[returnName] = returnAttr
	}

	return returns
}

// ---------------------------------------------------------------------------
// ReturnInfo
// ---------------------------------------------------------------------------

// ReturnInfo is the RETURN block for an info module.
// It always documents exactly two values: changed (always false) and resources
// (a list of zero or more resources matching the supplied filters, with a full
// contains schema derived from the resource's gettable properties).
type ReturnInfo struct {
	ResourceKind string                     // e.g. "AlloyDB.Cluster"
	Contains     map[string]*ReturnAttribute // schema of each item in the resources list
}

// NewReturnInfo creates a ReturnInfo for the given resource.
// All gettable properties are included in Contains with extended=false so that
// ResourceRef and immutability annotations are omitted from info module output.
func NewReturnInfo(resource *api.Resource) *ReturnInfo {
	return &ReturnInfo{
		ResourceKind: resource.Parent.Mmv1.Name + "." + resource.Mmv1.Name,
		Contains:     convertPropertiesToReturns(resource.Mmv1.GettableProperties(), false),
	}
}

// ToString serialises the return block to a YAML string.
func (r *ReturnInfo) ToString() string {
	returns := map[string]*ReturnAttribute{
		"changed": changedReturnAttribute("Whether any changes were made (always false for info modules)."),
		"resources": {
			Description: fmt.Sprintf(
				"List of %s resources matching the supplied filters. "+
					"May be empty, contain a single resource, or multiple resources.",
				r.ResourceKind,
			),
			Returned: "always",
			Type:     ReturnTypeList,
			Elements: ReturnTypeDict,
			Contains: r.Contains,
		},
	}
	return ToYAML(returns)
}

// determineReturnedCondition determines when a return value is returned based on property characteristics
func determineReturnedCondition(property *mmv1api.Type) string {
	if property == nil {
		return "success"
	}

	// Output-only properties are always returned when the resource exists
	if property.Output {
		return "success"
	}

	// Required properties are always returned
	if property.Required {
		return "always"
	}

	// Optional properties are returned when set
	return "when set"
}
