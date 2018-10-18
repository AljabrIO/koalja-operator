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
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	agentsv1alpha1 "github.com/AljabrIO/koalja-operator/pkg/apis/agents/v1alpha1"
	koaljav1alpha1 "github.com/AljabrIO/koalja-operator/pkg/apis/koalja/v1alpha1"
	"github.com/AljabrIO/koalja-operator/pkg/constants"
)

// Add creates a new Pipeline Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcilePipeline{
		Client:        mgr.GetClient(),
		scheme:        mgr.GetScheme(),
		eventRecorder: mgr.GetRecorder("controller"),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("pipeline-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to Pipeline
	err = c.Watch(&source.Kind{Type: &koaljav1alpha1.Pipeline{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch Deployments since we're launching a various Agent Deployments
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &koaljav1alpha1.Pipeline{},
	})
	if err != nil {
		return err
	}

	// Watch StatefulSets since we're launching a various Agent StatefulSets
	err = c.Watch(&source.Kind{Type: &appsv1.StatefulSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &koaljav1alpha1.Pipeline{},
	})
	if err != nil {
		return err
	}

	// Watch Services since we're launching a various Agent Services
	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &koaljav1alpha1.Pipeline{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcilePipeline{}

// ReconcilePipeline reconciles a Pipeline object
type ReconcilePipeline struct {
	client.Client
	scheme        *runtime.Scheme
	eventRecorder record.EventRecorder
}

// Reconcile reads that state of the cluster for a Pipeline object and makes changes based on the state read
// and what is in the Pipeline.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=v1,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=koalja.aljabr.io,resources=pipelines,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcilePipeline) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()

	// Fetch the Pipeline instance
	instance := &koaljav1alpha1.Pipeline{}
	if err := r.Get(ctx, request.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Validate the spec
	if err := instance.Spec.Validate(); err != nil {
		log.Printf("Pipeline is invalid: %s\n", err)
		r.eventRecorder.Eventf(instance, "Warning", "PipelineValidation", "Pipeline is not valid: %s", err)
		return reconcile.Result{}, nil
	} else {
		r.eventRecorder.Event(instance, "Norml", "PipelineValidation", "Pipeline is valid")
	}

	// Ensure pipeline agent is created
	var result reconcile.Result
	if lresult, err := r.ensurePipelineAgent(ctx, instance); err != nil {
		log.Printf("ensurePipelineAgent failed: %s\n", err)
		return lresult, err
	} else {
		result = MergeReconcileResult(result, lresult)
	}

	// Ensure link agents are created
	for _, l := range instance.Spec.Links {
		if lresult, err := r.ensureLinkAgent(ctx, instance, l); err != nil {
			log.Printf("ensureLinkAgent failed: %s\n", err)
			return lresult, err
		} else {
			result = MergeReconcileResult(result, lresult)
		}
	}

	// Ensure task agents are created
	for _, t := range instance.Spec.Tasks {
		if lresult, err := r.ensureTaskAgent(ctx, instance, t); err != nil {
			log.Printf("ensureTaskAgent failed: %s\n", err)
			return lresult, err
		} else {
			result = MergeReconcileResult(result, lresult)
		}
	}

	return result, nil
}

// ensurePipelineAgent ensures that a pipeline agent is launched for the given pipeline instance.
// +kubebuilder:rbac:groups=agents.aljabr.io,resources=pipelines,verbs=get;list;watch
func (r *ReconcilePipeline) ensurePipelineAgent(ctx context.Context, instance *koaljav1alpha1.Pipeline) (reconcile.Result, error) {
	// Search for pipeline agent resource
	var plAgentList agentsv1alpha1.PipelineList
	if err := r.List(ctx, &client.ListOptions{}, &plAgentList); err != nil {
		return reconcile.Result{}, err
	}
	if len(plAgentList.Items) == 0 {
		// No pipeline agent resource found
		log.Println("No Pipeline Agents found")
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: time.Second * 10,
		}, nil
	}
	agentCont := *plAgentList.Items[0].Spec.Container
	SetAgentContainerDefaults(&agentCont)
	SetContainerEnvVars(&agentCont, map[string]string{
		constants.EnvAPIPort:              strconv.Itoa(constants.AgentAPIPort),
		constants.EnvPipelineName:         instance.Name,
		constants.EnvEventRegistryAddress: CreateEventRegistryAddress(instance.Name, instance.Namespace),
		constants.EnvDNSName:              CreatePipelineAgentDNSName(instance.Name, instance.Namespace),
	})

	// Search for event registry resource
	var evtRegistryList agentsv1alpha1.EventRegistryList
	if err := r.List(ctx, &client.ListOptions{}, &evtRegistryList); err != nil {
		return reconcile.Result{}, err
	}
	if len(evtRegistryList.Items) == 0 {
		// No event registry resource found
		log.Println("No Event Registries found")
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: time.Second * 10,
		}, nil
	}
	evtRegistryCont := *evtRegistryList.Items[0].Spec.Container
	SetAgentContainerDefaults(&evtRegistryCont)
	SetContainerEnvVars(&evtRegistryCont, map[string]string{
		constants.EnvAPIPort:      strconv.Itoa(constants.EventRegistryAPIPort),
		constants.EnvPipelineName: instance.Name,
	})

	// Define the desired Deployment object for pipeline agent
	deplName := CreatePipelineAgentName(instance.Name)
	deplLabels := map[string]string{"statefulset": deplName}
	deploy := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deplName,
			Namespace: instance.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: deplLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: deplLabels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{agentCont, evtRegistryCont},
				},
			},
		},
	}
	if err := controllerutil.SetControllerReference(instance, deploy, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	{
		// Check if the pipeline agent Deployment already exists
		found := &appsv1.StatefulSet{}
		if err := r.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, found); err != nil && errors.IsNotFound(err) {
			log.Printf("Creating Pipeline Agent StatefulSet %s/%s\n", deploy.Namespace, deploy.Name)
			if err := r.Create(ctx, deploy); err != nil {
				log.Printf("Failed to create Pipeline Agent StatefulSet %s/%s: %s\n", deploy.Namespace, deploy.Name, err)
				return reconcile.Result{}, err
			}
		} else if err != nil {
			return reconcile.Result{}, err
		} else {
			// Update the found object and write the result back if there are any changes
			/*if !equality.Semantic.DeepEqual(deploy.Spec, found.Spec) {
				found.Spec = deploy.Spec
				log.Printf("Updating Pipeline Agent StatefulSet %s/%s\n", deploy.Namespace, deploy.Name)
				if err := r.Update(ctx, found); err != nil {
					return reconcile.Result{}, err
				}
			}
			TODO*/
		}
	}

	{
		// Define the desired Service object for pipeline agent
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deplName,
				Namespace: instance.Namespace,
			},
			Spec: corev1.ServiceSpec{
				Selector: deplLabels,
				Type:     corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{
					corev1.ServicePort{
						Name: "agent-api",
						Port: constants.AgentAPIPort,
					},
					corev1.ServicePort{
						Name: "event-registry-api",
						Port: constants.EventRegistryAPIPort,
					},
				},
			},
		}
		if err := controllerutil.SetControllerReference(instance, service, r.scheme); err != nil {
			return reconcile.Result{}, err
		}

		// Check if the pipeline agent Service already exists
		found := &corev1.Service{}
		if err := r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, found); err != nil && errors.IsNotFound(err) {
			log.Printf("Creating Pipeline Agent Service %s/%s\n", service.Namespace, service.Name)
			if err := r.Create(ctx, service); err != nil {
				log.Printf("Failed to create Pipeline Agent Service %s/%s: %s\n", service.Namespace, service.Name, err)
				return reconcile.Result{}, err
			}
		} else if err != nil {
			return reconcile.Result{}, err
		} else {
			// Update the found object and write the result back if there are any changes
			if !equality.Semantic.DeepEqual(service.Spec, found.Spec) {
				service.Spec.ClusterIP = found.Spec.ClusterIP
				found.Spec = service.Spec
				log.Printf("Updating Pipeline Agent Service %s/%s\n", service.Namespace, service.Name)
				if err := r.Update(ctx, found); err != nil {
					return reconcile.Result{}, err
				}
			}
		}
	}

	return reconcile.Result{}, nil
}

// ensureLinkAgent ensures that a link agent is launched for the given link in given pipeline instance.
// +kubebuilder:rbac:groups=agents.aljabr.io,resources=links,verbs=get;list;watch
func (r *ReconcilePipeline) ensureLinkAgent(ctx context.Context, instance *koaljav1alpha1.Pipeline, link koaljav1alpha1.LinkSpec) (reconcile.Result, error) {
	// Search for link agent resource
	var linkAgentList agentsv1alpha1.LinkList
	if err := r.List(ctx, &client.ListOptions{}, &linkAgentList); err != nil {
		return reconcile.Result{}, err
	}
	if len(linkAgentList.Items) == 0 {
		// No link agent resource found
		log.Println("No Link Agents found")
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: time.Second * 10,
		}, nil
	}
	c := *linkAgentList.Items[0].Spec.Container
	SetAgentContainerDefaults(&c)
	SetContainerEnvVars(&c, map[string]string{
		constants.EnvAPIPort:              strconv.Itoa(constants.AgentAPIPort),
		constants.EnvPipelineName:         instance.Name,
		constants.EnvLinkName:             link.Name,
		constants.EnvEventRegistryAddress: CreateEventRegistryAddress(instance.Name, instance.Namespace),
		constants.EnvDNSName:              CreateLinkAgentDNSName(instance.Name, link.Name, instance.Namespace),
	})

	// Define the desired StatefulSet object for link agent
	deplName := CreateLinkAgentName(instance.Name, link.Name)
	deplLabels := map[string]string{
		"statefulset": deplName,
		"link":        link.Name,
	}
	deploy := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deplName,
			Namespace: instance.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: deplLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: deplLabels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{c},
				},
			},
		},
	}
	if err := controllerutil.SetControllerReference(instance, deploy, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	{
		// Check if the link agent StatefulSet already exists
		found := &appsv1.StatefulSet{}
		if err := r.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, found); err != nil && errors.IsNotFound(err) {
			log.Printf("Creating Link Agent StatefulSet %s/%s\n", deploy.Namespace, deploy.Name)
			if err := r.Create(ctx, deploy); err != nil {
				log.Printf("Failed to create Link Agent StatefulSet %s/%s: %s\n", deploy.Namespace, deploy.Name, err)
				return reconcile.Result{}, err
			}
		} else if err != nil {
			return reconcile.Result{}, err
		} else {
			// Update the found object and write the result back if there are any changes
			/*if !equality.Semantic.DeepEqual(deploy.Spec, found.Spec) {
				found.Spec = deploy.Spec
				log.Printf("Updating Link Agent StatefulSet %s/%s\n", deploy.Namespace, deploy.Name)
				if err := r.Update(ctx, found); err != nil {
					return reconcile.Result{}, err
				}
			}
			TODO
			*/
		}
	}

	{
		// Define the desired Service object for link agent
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deplName,
				Namespace: instance.Namespace,
			},
			Spec: corev1.ServiceSpec{
				Selector: deplLabels,
				Type:     corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{
					corev1.ServicePort{
						Name: "api",
						Port: constants.AgentAPIPort,
					},
				},
			},
		}
		if err := controllerutil.SetControllerReference(instance, service, r.scheme); err != nil {
			return reconcile.Result{}, err
		}

		// Check if the link agent Service already exists
		found := &corev1.Service{}
		if err := r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, found); err != nil && errors.IsNotFound(err) {
			log.Printf("Creating Link Agent Service %s/%s\n", service.Namespace, service.Name)
			if err := r.Create(ctx, service); err != nil {
				log.Printf("Failed to create Link Agent Service %s/%s: %s\n", service.Namespace, service.Name, err)
				return reconcile.Result{}, err
			}
		} else if err != nil {
			return reconcile.Result{}, err
		} else {
			// Update the found object and write the result back if there are any changes
			if !equality.Semantic.DeepEqual(service.Spec, found.Spec) {
				service.Spec.ClusterIP = found.Spec.ClusterIP
				found.Spec = service.Spec
				log.Printf("Updating Link Agent Service %s/%s\n", service.Namespace, service.Name)
				if err := r.Update(ctx, found); err != nil {
					return reconcile.Result{}, err
				}
			}
		}
	}

	return reconcile.Result{}, nil
}

// ensureTaskAgent ensures that a task agent is launched for the given task in given pipeline instance.
// +kubebuilder:rbac:groups=agents.aljabr.io,resources=tasks,verbs=get;list;watch
func (r *ReconcilePipeline) ensureTaskAgent(ctx context.Context, instance *koaljav1alpha1.Pipeline, task koaljav1alpha1.TaskSpec) (reconcile.Result, error) {
	// Search for task agent resource
	var taskAgentList agentsv1alpha1.TaskList
	if err := r.List(ctx, &client.ListOptions{}, &taskAgentList); err != nil {
		return reconcile.Result{}, err
	}
	if len(taskAgentList.Items) == 0 {
		// No task agent resource found
		log.Println("No Task Agents found")
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: time.Second * 10,
		}, nil
	}
	c := *taskAgentList.Items[0].Spec.Container
	SetAgentContainerDefaults(&c)
	SetContainerEnvVars(&c, map[string]string{
		constants.EnvPipelineName:         instance.Name,
		constants.EnvTaskName:             task.Name,
		constants.EnvEventRegistryAddress: CreateEventRegistryAddress(instance.Name, instance.Namespace),
	})

	// Define the desired StatefulSet object for task agent
	deplName := CreateTaskAgentName(instance.Name, task.Name)
	deplLabels := map[string]string{
		"statefulset": deplName,
		"task":        task.Name,
	}
	// Create annotations to pass address of input links
	annotations := make(map[string]string)
	for _, tis := range task.Inputs {
		annKey := constants.CreateInputLinkAddressAnnotationName(tis.Name)
		ref := task.Name + "/" + tis.Name
		link, found := instance.Spec.LinkByDestinationRef(ref)
		if !found {
			log.Printf("No link found with DestinationRef '%s'\n", ref)
			return reconcile.Result{}, fmt.Errorf("No link found with DestinationRef '%s'", ref)
		}
		annotations[annKey] = CreateLinkAgentEventSourceAddress(instance.Name, link.Name, instance.Namespace)
	}
	// Create annotations to pass addresses of output links
	for _, tos := range task.Outputs {
		annKey := constants.CreateOutputLinkAddressesAnnotationName(tos.Name)
		ref := task.Name + "/" + tos.Name
		links := instance.Spec.LinksBySourceRef(ref)
		if len(links) > 0 {
			addresses := make([]string, 0, len(links))
			for _, link := range links {
				addresses = append(addresses, CreateLinkAgentEventPublisherAddress(instance.Name, link.Name, instance.Namespace))
			}
			annotations[annKey] = strings.Join(addresses, ",")
		} else {
			// Task output is not connected. Connect it to the pipeline agent
			annotations[annKey] = CreatePipelineAgentEventPublisherAddress(instance.Name, instance.Namespace)
		}
	}
	deploy := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deplName,
			Namespace: instance.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: deplLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      deplLabels,
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{c},
				},
			},
		},
	}
	if err := controllerutil.SetControllerReference(instance, deploy, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	{
		// Check if the task agent StatefulSet already exists
		found := &appsv1.StatefulSet{}
		if err := r.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, found); err != nil && errors.IsNotFound(err) {
			log.Printf("Creating Task Agent StatefulSet %s/%s\n", deploy.Namespace, deploy.Name)
			if err := r.Create(ctx, deploy); err != nil {
				log.Printf("Failed to create Task Agent StatefulSet %s/%s: %s\n", deploy.Namespace, deploy.Name, err)
				return reconcile.Result{}, err
			}
		} else if err != nil {
			return reconcile.Result{}, err
		} else {
			// Update the found object and write the result back if there are any changes
			/*if !equality.Semantic.DeepEqual(deploy.Spec, found.Spec) {
				found.Spec = deploy.Spec
				log.Printf("Updating Task Agent StatefulSet %s/%s\n", deploy.Namespace, deploy.Name)
				if err := r.Update(ctx, found); err != nil {
					return reconcile.Result{}, err
				}
			}
			TODO
			*/
		}
	}

	return reconcile.Result{}, nil
}
