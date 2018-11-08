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

package contour

import (
	"context"

	contour "github.com/heptio/contour/apis/contour/v1beta1"
	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/AljabrIO/koalja-operator/pkg/controller/pipeline"
	"github.com/AljabrIO/koalja-operator/pkg/util"
)

// New creates a new Contour network reconciler.
func New(mgr manager.Manager) (pipeline.NetworkReconciler, error) {
	return &reconciler{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}, nil
}

type reconciler struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile network resources for the given pipeline.
func (r *reconciler) Reconcile(ctx context.Context, log zerolog.Logger, req pipeline.NetworkReconcileRequest) (reconcile.Result, error) {
	// Create IngressRoute
	ingressRt := &contour.IngressRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Pipeline.GetName(),
			Namespace: req.Pipeline.GetNamespace(),
		},
		Spec: contour.IngressRouteSpec{
			VirtualHost: &contour.VirtualHost{
				Fqdn: req.Pipeline.Status.Domain,
				// TODO TLS
			},
		},
	}
	if req.Frontend.Route != nil {
		ingressRt.Spec.Routes = append(ingressRt.Spec.Routes, contour.Route{
			Match: "/",
			Services: []contour.Service{
				contour.Service{
					Name: req.Frontend.ServiceName,
					Port: req.Frontend.Route.Port,
				},
			},
			EnableWebsockets: req.Frontend.Route.EnableWebsockets,
			PrefixRewrite:    req.Frontend.Route.PrefixRewrite,
		})
	}
	for _, tx := range req.TaskExecutors {
		cRoute := contour.Route{
			Match: "/" + tx.TaskName,
		}
		for _, r := range tx.Routes {
			cRoute.Services = append(cRoute.Services, contour.Service{
				Name: tx.ServiceName,
				Port: r.Port,
			})
			cRoute.EnableWebsockets = cRoute.EnableWebsockets || r.EnableWebsockets
			cRoute.PrefixRewrite = r.PrefixRewrite
		}
		ingressRt.Spec.Routes = append(ingressRt.Spec.Routes, cRoute)
	}
	if err := controllerutil.SetControllerReference(req.Pipeline, ingressRt, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Ensure ingress exists
	if err := util.EnsureIngressRoute(ctx, log, r.Client, ingressRt, "Pipeline Ingress Route"); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// SetPodAnnotations sets relevant annotation on the metadata of a pod.
func (r *reconciler) SetPodAnnotations(metadata *metav1.ObjectMeta) {
	// Nothing needed
}
