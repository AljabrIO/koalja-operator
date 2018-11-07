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

package util

import (
	"context"

	contour "github.com/heptio/contour/apis/contour/v1beta1"
	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EnsureIngressRoute creates of updates a IngressRoute.
func EnsureIngressRoute(ctx context.Context, log zerolog.Logger, client client.Client, route *contour.IngressRoute, description string) error {
	// Check if IngressRoute already exists
	found := &contour.IngressRoute{}
	if err := client.Get(ctx, types.NamespacedName{Name: route.Name, Namespace: route.Namespace}, found); err != nil && errors.IsNotFound(err) {
		log.Info().Msgf("Creating %s", description)
		if err := client.Create(ctx, route); err != nil {
			log.Error().Err(err).Msgf("Failed to create %s", description)
			return err
		}
	} else if err != nil {
		return err
	} else {
		// Update the found object and write the result back if there are any changes
		if diff := IngressRouteEqual(*route, *found); len(diff) > 0 {
			found.Spec = route.Spec
			log.Info().Interface("diff", diff).Msgf("Updating %s", description)
			if err := client.Update(ctx, found); err != nil {
				return err
			}
		}
	}

	return nil
}
