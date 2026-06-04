// Copyright 2025 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0

package ansible

import (
	"strings"

	mmv1api "github.com/GoogleCloudPlatform/magic-modules/mmv1/api"
	"github.com/rs/zerolog/log"
)

type OperationConfigs struct {
	BaseUri       string
	CollectionUrl string
	Configs       map[string]*OperationConfig
}

type OperationConfig struct {
	UriTemplate      string `json:"uri"`
	AsyncUriTemplate string `json:"async_uri"`
	Verb             string `json:"verb"`
	TimeoutMinutes   int    `json:"timeout_minutes"`
}

func escapeCurlyBraces(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "{{", "{"), "}}", "}")
}

func NewOperationConfigsFromMmv1(mmv1 *mmv1api.Resource) *OperationConfigs {
	opConfig := &OperationConfigs{
		BaseUri:       escapeCurlyBraces(mmv1.BaseUrl),
		CollectionUrl: escapeCurlyBraces(mmv1.CollectionUrl()),
	}

	ops := map[string]*OperationConfig{}
	timeouts := mmv1.GetTimeouts()
	defaultVerbs := map[string]string{
		"read":   "GET",
		"create": "POST",
		"update": "PUT",
		"delete": "DELETE",
	}

	// Helper function to get verb or default
	getVerb := func(mmv1Verb, operation string) string {
		if mmv1Verb != "" {
			return mmv1Verb
		}
		return defaultVerbs[operation]
	}

	ops["read"] = &OperationConfig{
		UriTemplate:      escapeCurlyBraces(mmv1.SelfLinkUri()),
		Verb:             getVerb(mmv1.ReadVerb, "read"),
		AsyncUriTemplate: "",
	}
	ops["create"] = &OperationConfig{
		UriTemplate:      escapeCurlyBraces(mmv1.CreateUri()),
		Verb:             getVerb(mmv1.CreateVerb, "create"),
		TimeoutMinutes:   timeouts.InsertMinutes,
		AsyncUriTemplate: "",
	}
	ops["update"] = &OperationConfig{
		UriTemplate:      escapeCurlyBraces(mmv1.UpdateUri()),
		Verb:             getVerb(mmv1.UpdateVerb, "update"),
		TimeoutMinutes:   timeouts.UpdateMinutes,
		AsyncUriTemplate: "",
	}
	ops["delete"] = &OperationConfig{
		UriTemplate:      escapeCurlyBraces(mmv1.DeleteUri()),
		Verb:             getVerb(mmv1.DeleteVerb, "delete"),
		TimeoutMinutes:   timeouts.DeleteMinutes,
		AsyncUriTemplate: "",
	}

	async := mmv1.GetAsync()
	log.Debug().Msgf("async: %v", async)
	if async != nil {
		for _, action := range async.Actions {
			if async.Operation == nil || async.Operation.BaseUrl == "" {
				ops[strings.ToLower(action)].AsyncUriTemplate = "{op_id}"
			} else {
				ops[strings.ToLower(action)].AsyncUriTemplate = escapeCurlyBraces(async.Operation.BaseUrl)
			}
		}
	}

	opConfig.Configs = ops

	log.Debug().Msgf("operation configs: %v", opConfig)

	return opConfig
}
