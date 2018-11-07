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

	agentsapi "github.com/AljabrIO/koalja-operator/pkg/apis/agents/v1alpha1"
	koaljav1alpha1 "github.com/AljabrIO/koalja-operator/pkg/apis/koalja/v1alpha1"
	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	globalNetworkReconciler NetworkReconciler
)

// NetworkReconciler abstracts away reconcilition for different network
// environments.
type NetworkReconciler interface {
	// Reconcile network resources for the given pipeline.
	Reconcile(ctx context.Context, log zerolog.Logger, request NetworkReconcileRequest) (reconcile.Result, error)
	// SetPodAnnotations sets relevant annotation on the metadata of a pod.
	SetPodAnnotations(metadata *metav1.ObjectMeta)
}

// NetworkReconcileRequest holds all the arguments for a Reconcile request.
type NetworkReconcileRequest struct {
	Request       reconcile.Request
	Pipeline      *koaljav1alpha1.Pipeline
	Frontend      Frontend
	TaskExecutors []TaskExecutor
}

// Frontend network info
type Frontend struct {
	// ServiceName is the name of the k8s Service that has been created for the frontend
	ServiceName string
	// Routes that need to be configured for the frontend
	Route *agentsapi.NetworkRoute
}

// TaskExecutor holds information on a selector task executor.
type TaskExecutor struct {
	// TaskName is the name of the task
	TaskName string
	// ServiceName is the name of the k8s Service that has been created for this executor
	ServiceName string
	// Routes that need to be configured for the executor
	Routes []agentsapi.NetworkRoute
}

// SetGlobalNetworkReconciler sets the global network reconciler
func SetGlobalNetworkReconciler(r NetworkReconciler) {
	globalNetworkReconciler = r
}
