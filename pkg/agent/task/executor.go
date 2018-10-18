//
// Copyright © 2018 Aljabr, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package task

import (
	"context"
	"fmt"
	"strings"

	koalja "github.com/AljabrIO/koalja-operator/pkg/apis/koalja/v1alpha1"
	"github.com/AljabrIO/koalja-operator/pkg/constants"
	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Executor describes the API implemented by a task executor
type Executor interface {
	// Execute on the task with the given snapshot as input.
	Execute(context.Context, *InputSnapshot) error
}

// NewExecutor initializes a new Executor.
func NewExecutor(log zerolog.Logger, client client.Client, spec *koalja.TaskSpec, pod *corev1.Pod) (Executor, error) {
	// Get output addresses
	outputAddressesMap := make(map[string][]string)
	for _, tos := range spec.Outputs {
		annKey := constants.CreateOutputLinkAddressesAnnotationName(tos.Name)
		addresses := pod.GetAnnotations()[annKey]
		if addresses == "" {
			return nil, fmt.Errorf("No output addresses annotation found for output '%s'", tos.Name)
		}
		outputAddressesMap[tos.Name] = strings.Split(addresses, ",")
	}

	return &executor{
		Client:             client,
		log:                log,
		spec:               spec,
		outputAddressesMap: outputAddressesMap,
	}, nil
}

type executor struct {
	client.Client
	log                zerolog.Logger
	spec               *koalja.TaskSpec
	outputAddressesMap map[string][]string
}

// Execute on the task with the given snapshot as input.
func (e *executor) Execute(context.Context, *InputSnapshot) error {
	// TODO
	return nil
}
