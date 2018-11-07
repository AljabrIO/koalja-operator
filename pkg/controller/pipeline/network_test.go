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

	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type testNetworkReconciler struct {
}

// Reconcile network resources for the given pipeline.
func (r *testNetworkReconciler) Reconcile(ctx context.Context, log zerolog.Logger, request NetworkReconcileRequest) (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

// SetPodAnnotations sets relevant annotation on the metadata of a pod.
func (r *testNetworkReconciler) SetPodAnnotations(metadata *metav1.ObjectMeta) {}
