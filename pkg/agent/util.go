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

package agent

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	koaljav1alpha1 "github.com/AljabrIO/koalja-operator/pkg/apis/koalja/v1alpha1"
	"github.com/AljabrIO/koalja-operator/pkg/constants"
)

// GetPipeline loads the pipeline identified by a name in the environment.
func GetPipeline(ctx context.Context, kapi client.Client) (*koaljav1alpha1.Pipeline, error) {
	name, err := constants.GetPipelineName()
	if err != nil {
		return nil, err
	}
	ns, err := constants.GetNamespace()
	if err != nil {
		return nil, err
	}
	instance := &koaljav1alpha1.Pipeline{}
	key := types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}
	if err := kapi.Get(ctx, key, instance); err != nil {
		return nil, err
	}
	return instance, nil
}
