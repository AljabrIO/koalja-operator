/*
Copyright 2018 Aljabr Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pipeline

import (
	"context"

	"github.com/AljabrIO/koalja-operator/pkg/constants"

	agentsapi "github.com/AljabrIO/koalja-operator/pkg/apis/agents/v1alpha1"
	"github.com/rs/zerolog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// selectPipelineAgent selects the pipeline agent to use for the given namespace.
func selectPipelineAgent(ctx context.Context, log zerolog.Logger, c client.Client, ns string) (*agentsapi.Pipeline, error) {
	var list agentsapi.PipelineList
	if err := c.List(ctx, &client.ListOptions{Namespace: ns}, &list); err != nil {
		return nil, err
	}
	if len(list.Items) == 0 {
		// No agents found in specific namespace, fallback to system namespace
		if err := c.List(ctx, &client.ListOptions{Namespace: constants.NamespaceKoaljaSystem}, &list); err != nil {
			return nil, err
		}
	}
	if len(list.Items) == 0 {
		// No agents found
		log.Warn().Msg("No Pipeline Agents found")
		return nil, nil
	}
	return &list.Items[0], nil
}

// selectLinkAgent selects the link agent to use for the given namespace.
func selectLinkAgent(ctx context.Context, log zerolog.Logger, c client.Client, ns string) (*agentsapi.Link, error) {
	var list agentsapi.LinkList
	if err := c.List(ctx, &client.ListOptions{Namespace: ns}, &list); err != nil {
		return nil, err
	}
	if len(list.Items) == 0 {
		// No agents found in specific namespace, fallback to system namespace
		if err := c.List(ctx, &client.ListOptions{Namespace: constants.NamespaceKoaljaSystem}, &list); err != nil {
			return nil, err
		}
	}
	if len(list.Items) == 0 {
		// No agents found
		log.Warn().Msg("No Link Agents found")
		return nil, nil
	}
	return &list.Items[0], nil
}

// selectTaskAgent selects the task agent to use for the given namespace.
func selectTaskAgent(ctx context.Context, log zerolog.Logger, c client.Client, ns string) (*agentsapi.Task, error) {
	var list agentsapi.TaskList
	if err := c.List(ctx, &client.ListOptions{Namespace: ns}, &list); err != nil {
		return nil, err
	}
	if len(list.Items) == 0 {
		// No agents found in specific namespace, fallback to system namespace
		if err := c.List(ctx, &client.ListOptions{Namespace: constants.NamespaceKoaljaSystem}, &list); err != nil {
			return nil, err
		}
	}
	if len(list.Items) == 0 {
		// No agents found
		log.Warn().Msg("No Task Agents found")
		return nil, nil
	}
	return &list.Items[0], nil
}

// selectAnnotatedValueRegistry selects the annotated value registry to use for the given namespace.
func selectAnnotatedValueRegistry(ctx context.Context, log zerolog.Logger, c client.Client, defaultServiceName, ns string) (*agentsapi.AnnotatedValueRegistry, error) {
	var list1, list2 agentsapi.AnnotatedValueRegistryList
	if err := c.List(ctx, &client.ListOptions{Namespace: ns}, &list1); err != nil {
		return nil, err
	}
	if err := c.List(ctx, &client.ListOptions{Namespace: constants.NamespaceKoaljaSystem}, &list2); err != nil {
		return nil, err
	}
	all := append(list1.Items, list2.Items...)
	if defaultServiceName != "" {
		for _, x := range all {
			if x.Name == defaultServiceName {
				return &x, nil
			}
		}
	} else if len(all) > 0 {
		return &all[0], nil
	}

	// No (matching) agents found
	log.Warn().Msg("No Annotated Value Registries found")
	return nil, nil
}

// selectTaskExecutor selects the task executor to use for the given type and namespace.
func selectTaskExecutor(ctx context.Context, log zerolog.Logger, c client.Client, taskType, ns string) (*agentsapi.TaskExecutor, error) {
	var list agentsapi.TaskExecutorList
	if err := c.List(ctx, &client.ListOptions{Namespace: ns}, &list); err != nil {
		return nil, err
	}
	for _, entry := range list.Items {
		if string(entry.Spec.Type) == taskType {
			// Found
			return &entry, nil
		}
	}
	// No matching executors found in specific namespace, fallback to system namespace
	if err := c.List(ctx, &client.ListOptions{Namespace: constants.NamespaceKoaljaSystem}, &list); err != nil {
		return nil, err
	}
	for _, entry := range list.Items {
		if string(entry.Spec.Type) == string(taskType) {
			// Found
			return &entry, nil
		}
	}
	// No matching executor found
	log.Warn().Msgf("No TaskExecutor of type '%s' found", taskType)
	return nil, nil
}
