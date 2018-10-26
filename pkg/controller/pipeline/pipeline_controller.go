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
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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
	"github.com/AljabrIO/koalja-operator/pkg/util"
	"github.com/rs/zerolog"
)

// Add creates a new Pipeline Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcilePipeline{
		log:           util.MustCreateLogger(),
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
	log zerolog.Logger
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
	log := r.log.With().
		Str("name", request.Name).
		Str("namespace", request.Namespace).
		Logger()
	log.Debug().Msg("reconcile")

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
		log.Warn().Err(err).Msg("Pipeline is invalid")
		r.eventRecorder.Eventf(instance, "Warning", "PipelineValidation", "Pipeline is not valid: %s", err)
		return reconcile.Result{}, nil
	} else {
		r.eventRecorder.Event(instance, "Normal", "PipelineValidation", "Pipeline is valid")
	}

	// Ensure pipeline agent is created
	result := reconcile.Result{
		Requeue:      true,
		RequeueAfter: time.Minute,
	}
	if lresult, err := r.ensurePipelineAgent(ctx, instance); err != nil {
		log.Error().Err(err).Msg("ensurePipelineAgent failed")
		return lresult, err
	} else {
		result = MergeReconcileResult(result, lresult)
	}

	// Ensure link agents are created
	for _, l := range instance.Spec.Links {
		if lresult, err := r.ensureLinkAgent(ctx, instance, l); err != nil {
			log.Error().Err(err).Msg("ensureLinkAgent failed")
			return lresult, err
		} else {
			result = MergeReconcileResult(result, lresult)
		}
	}

	// Ensure task agents are created
	for _, t := range instance.Spec.Tasks {
		if lresult, err := r.ensureTaskAgent(ctx, instance, t); err != nil {
			log.Error().Err(err).Msg("ensureTaskAgent failed")
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
		r.log.Warn().Msg("No Pipeline Agents found")
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: time.Second * 10,
		}, nil
	}
	agentCont := *plAgentList.Items[0].Spec.Container
	SetAgentContainerDefaults(&agentCont, true)
	SetContainerEnvVars(&agentCont, map[string]string{
		constants.EnvAPIPort:              strconv.Itoa(constants.AgentAPIPort),
		constants.EnvAPIHTTPPort:          strconv.Itoa(constants.AgentAPIHTTPPort),
		constants.EnvPipelineName:         instance.Name,
		constants.EnvAgentRegistryAddress: CreateAgentRegistryAddress(instance.Name, instance.Namespace),
		constants.EnvEventRegistryAddress: net.JoinHostPort("localhost", strconv.Itoa(constants.EventRegistryAPIPort)),
		constants.EnvDNSName:              CreatePipelineAgentDNSName(instance.Name, instance.Namespace),
	})

	// Search for event registry resource
	var evtRegistryList agentsv1alpha1.EventRegistryList
	if err := r.List(ctx, &client.ListOptions{}, &evtRegistryList); err != nil {
		return reconcile.Result{}, err
	}
	if len(evtRegistryList.Items) == 0 {
		// No event registry resource found
		r.log.Warn().Msg("No Event Registries found")
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: time.Second * 10,
		}, nil
	}
	evtRegistryCont := *evtRegistryList.Items[0].Spec.Container
	SetEventRegistryContainerDefaults(&evtRegistryCont)
	SetContainerEnvVars(&evtRegistryCont, map[string]string{
		constants.EnvAPIPort:      strconv.Itoa(constants.EventRegistryAPIPort),
		constants.EnvPipelineName: instance.Name,
	})

	// Define the desired Deployment object for pipeline agent
	deplName := CreatePipelineAgentName(instance.Name)
	createDeplLabels := func() map[string]string {
		return map[string]string{
			"statefulset": deplName,
		}
	}
	deploy := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deplName,
			Namespace: instance.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: createDeplLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: createDeplLabels()},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{agentCont, evtRegistryCont},
				},
			},
		},
	}
	log := r.log.With().
		Str("name", deploy.Name).
		Str("namespace", deploy.Namespace).
		Logger()
	if err := controllerutil.SetControllerReference(instance, deploy, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	{
		// Check if the pipeline agent Deployment already exists
		found := &appsv1.StatefulSet{}
		if err := r.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, found); err != nil && errors.IsNotFound(err) {
			log.Info().Msg("Creating Pipeline Agent StatefulSet")
			if err := r.Create(ctx, deploy); err != nil {
				log.Error().Err(err).Msg("Failed to create Pipeline Agent StatefulSet")
				return reconcile.Result{}, err
			}
		} else if err != nil {
			return reconcile.Result{}, err
		} else {
			// Update the found object and write the result back if there are any changes
			if diff := util.StatefulSetEqual(*deploy, *found); len(diff) > 0 {
				found.Spec = deploy.Spec
				log.Info().Interface("diff", diff).Msg("Updating Pipeline Agent StatefulSet")
				if err := r.Update(ctx, found); err != nil {
					return reconcile.Result{}, err
				}
			}
		}
	}

	{
		// Define the desired Service object for pipeline agent
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deplName,
				Namespace: instance.Namespace,
				Labels:    createDeplLabels(),
			},
			Spec: corev1.ServiceSpec{
				Selector: createDeplLabels(),
				Type:     corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{
					corev1.ServicePort{
						Name:       "grpc-agent-api",
						Port:       constants.AgentAPIPort,
						TargetPort: intstr.FromInt(constants.AgentAPIPort),
						Protocol:   corev1.ProtocolTCP,
					},
					corev1.ServicePort{
						Name:       "http-agent-api",
						Port:       constants.AgentAPIHTTPPort,
						TargetPort: intstr.FromInt(constants.AgentAPIHTTPPort),
						Protocol:   corev1.ProtocolTCP,
					},
					corev1.ServicePort{
						Name:       "grpc-event-registry-api",
						Port:       constants.EventRegistryAPIPort,
						TargetPort: intstr.FromInt(constants.EventRegistryAPIPort),
						Protocol:   corev1.ProtocolTCP,
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
			log.Info().Msg("Creating Pipeline Agent Service")
			if err := r.Create(ctx, service); err != nil {
				log.Error().Err(err).Msg("Failed to create Pipeline Agent Service")
				return reconcile.Result{}, err
			}
		} else if err != nil {
			return reconcile.Result{}, err
		} else {
			// Update the found object and write the result back if there are any changes
			if diff := util.ServiceEqual(*service, *found); len(diff) > 0 {
				service.Spec.ClusterIP = found.Spec.ClusterIP
				found.Spec = service.Spec
				log.Info().Interface("diff", diff).Msg("Updating Pipeline Agent Service")
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
		r.log.Warn().Msg("No Link Agents found")
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: time.Second * 10,
		}, nil
	}
	c := *linkAgentList.Items[0].Spec.Container
	SetAgentContainerDefaults(&c, false)
	SetContainerEnvVars(&c, map[string]string{
		constants.EnvAPIPort:              strconv.Itoa(constants.AgentAPIPort),
		constants.EnvPipelineName:         instance.Name,
		constants.EnvLinkName:             link.Name,
		constants.EnvAgentRegistryAddress: CreateAgentRegistryAddress(instance.Name, instance.Namespace),
		constants.EnvEventRegistryAddress: CreateEventRegistryAddress(instance.Name, instance.Namespace),
		constants.EnvDNSName:              CreateLinkAgentDNSName(instance.Name, link.Name, instance.Namespace),
	})

	// Define the desired StatefulSet object for link agent
	deplName := CreateLinkAgentName(instance.Name, link.Name)
	createDeplLabels := func() map[string]string {
		return map[string]string{
			"statefulset": deplName,
			"link":        link.Name,
		}
	}
	deploy := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deplName,
			Namespace: instance.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: createDeplLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: createDeplLabels()},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{c},
				},
			},
		},
	}
	log := r.log.With().
		Str("name", deploy.Name).
		Str("namespace", deploy.Namespace).
		Logger()
	if err := controllerutil.SetControllerReference(instance, deploy, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	{
		// Check if the link agent StatefulSet already exists
		found := &appsv1.StatefulSet{}
		if err := r.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, found); err != nil && errors.IsNotFound(err) {
			log.Info().Msg("Creating Link Agent StatefulSet")
			if err := r.Create(ctx, deploy); err != nil {
				log.Error().Err(err).Msg("Failed to create Link Agent StatefulSet")
				return reconcile.Result{}, err
			}
		} else if err != nil {
			return reconcile.Result{}, err
		} else {
			// Update the found object and write the result back if there are any changes
			if diff := util.StatefulSetEqual(*deploy, *found); len(diff) > 0 {
				found.Spec = deploy.Spec
				log.Info().Interface("diff", diff).Msg("Updating Link Agent StatefulSet")
				if err := r.Update(ctx, found); err != nil {
					return reconcile.Result{}, err
				}
			}
		}
	}

	{
		// Define the desired Service object for link agent
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deplName,
				Namespace: instance.Namespace,
				Labels:    createDeplLabels(),
			},
			Spec: corev1.ServiceSpec{
				Selector: createDeplLabels(),
				Type:     corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{
					corev1.ServicePort{
						Name:       "grpc-api",
						Port:       constants.AgentAPIPort,
						TargetPort: intstr.FromInt(constants.AgentAPIPort),
						Protocol:   corev1.ProtocolTCP,
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
			log.Info().Msg("Creating Link Agent Service")
			if err := r.Create(ctx, service); err != nil {
				log.Error().Err(err).Msg("Failed to create Link Agent Service")
				return reconcile.Result{}, err
			}
		} else if err != nil {
			return reconcile.Result{}, err
		} else {
			// Update the found object and write the result back if there are any changes
			if diff := util.ServiceEqual(*service, *found); len(diff) > 0 {
				service.Spec.ClusterIP = found.Spec.ClusterIP
				found.Spec = service.Spec
				log.Info().Interface("diff", diff).Msg("Updating Link Agent Service")
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
// +kubebuilder:rbac:groups=agents.aljabr.io,resources=taskexecutors,verbs=get;list;watch
func (r *ReconcilePipeline) ensureTaskAgent(ctx context.Context, instance *koaljav1alpha1.Pipeline, task koaljav1alpha1.TaskSpec) (reconcile.Result, error) {
	// Search for FileSystem service
	var svcList corev1.ServiceList
	labelSel, err := labels.Parse(constants.LabelServiceType + "=" + constants.ServiceTypeFilesystem)
	if err != nil {
		return reconcile.Result{}, err
	}
	if err := r.List(ctx, &client.ListOptions{
		LabelSelector: labelSel,
	}, &svcList); err != nil {
		return reconcile.Result{}, err
	}
	if len(svcList.Items) == 0 {
		// No task agent resource found
		r.log.Warn().Msg("No FileSystem service found")
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: time.Second * 10,
		}, nil
	}
	filesystemServiceAddress := CreateServiceAddress(svcList.Items[0])

	// Search for matching taskexecutor (if needed)
	var annTaskExecutorContainer string
	if task.Type != "" {
		var taskExecList agentsv1alpha1.TaskExecutorList
		if err := r.List(ctx, &client.ListOptions{Namespace: instance.Namespace}, &taskExecList); err != nil {
			return reconcile.Result{}, err
		}
		found := false
		for _, entry := range taskExecList.Items {
			if string(entry.Spec.Type) == string(task.Type) {
				// Found
				found = true
				encoded, err := json.Marshal(entry.Spec.Container)
				if err != nil {
					return reconcile.Result{}, err
				}
				annTaskExecutorContainer = string(encoded)
				break
			}
		}
		if !found {
			r.eventRecorder.Eventf(instance, "Warning", "PipelineValidation", "No TaskExecutor of type '%s' found for task '%s'", task.Type, task.Name)
			r.log.Warn().Msgf("No TaskExecutor of type '%s' found", task.Type)
			return reconcile.Result{
				Requeue:      true,
				RequeueAfter: time.Second * 10,
			}, nil
		}
	}

	// Search for task agent resource
	var taskAgentList agentsv1alpha1.TaskList
	if err := r.List(ctx, &client.ListOptions{}, &taskAgentList); err != nil {
		return reconcile.Result{}, err
	}
	if len(taskAgentList.Items) == 0 {
		// No task agent resource found
		r.log.Warn().Msg("No Task Agents found")
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: time.Second * 10,
		}, nil
	}
	c := *taskAgentList.Items[0].Spec.Container
	SetAgentContainerDefaults(&c, false)
	SetContainerEnvVars(&c, map[string]string{
		constants.EnvAPIPort:              strconv.Itoa(constants.AgentAPIPort),
		constants.EnvPipelineName:         instance.Name,
		constants.EnvTaskName:             task.Name,
		constants.EnvAgentRegistryAddress: CreateAgentRegistryAddress(instance.Name, instance.Namespace),
		constants.EnvEventRegistryAddress: CreateEventRegistryAddress(instance.Name, instance.Namespace),
		constants.EnvFileSystemAddress:    filesystemServiceAddress,
		constants.EnvDNSName:              CreateTaskAgentDNSName(instance.Name, task.Name, instance.Namespace),
	})

	// Define the desired StatefulSet object for task agent
	deplName := CreateTaskAgentName(instance.Name, task.Name)
	createDeplLabels := func() map[string]string {
		return map[string]string{
			"statefulset": deplName,
			"task":        task.Name,
		}
	}
	// Create annotations to pass address of input links
	annotations := make(map[string]string)
	for _, tis := range task.Inputs {
		annKey := constants.CreateInputLinkAddressAnnotationName(tis.Name)
		ref := task.Name + "/" + tis.Name
		link, found := instance.Spec.LinkByDestinationRef(ref)
		if !found {
			r.log.Error().Str("ref", ref).Msg("No link found for DestinationRef")
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
	// Create annotation containing task executor container (if any)
	if annTaskExecutorContainer != "" {
		annotations[constants.AnnTaskExecutorContainer] = annTaskExecutorContainer
	}
	deploy := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deplName,
			Namespace: instance.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: createDeplLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      createDeplLabels(),
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{c},
				},
			},
		},
	}
	log := r.log.With().
		Str("name", deploy.Name).
		Str("namespace", deploy.Namespace).
		Logger()
	if err := controllerutil.SetControllerReference(instance, deploy, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	{
		// Check if the task agent StatefulSet already exists
		found := &appsv1.StatefulSet{}
		if err := r.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, found); err != nil && errors.IsNotFound(err) {
			log.Info().Msg("Creating Task Agent StatefulSet")
			if err := r.Create(ctx, deploy); err != nil {
				log.Error().Err(err).Msg("Failed to create Task Agent StatefulSet")
				return reconcile.Result{}, err
			}
		} else if err != nil {
			return reconcile.Result{}, err
		} else {
			// Update the found object and write the result back if there are any changes
			if diff := util.StatefulSetEqual(*deploy, *found); len(diff) > 0 {
				found.Spec = deploy.Spec
				log.Info().Interface("diff", diff).Msg("Updating Task Agent StatefulSet")
				if err := r.Update(ctx, found); err != nil {
					return reconcile.Result{}, err
				}
			}
		}
	}

	{
		// Define the desired Service object for task agent
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deplName,
				Namespace: instance.Namespace,
				Labels:    createDeplLabels(),
			},
			Spec: corev1.ServiceSpec{
				Selector: createDeplLabels(),
				Type:     corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{
					corev1.ServicePort{
						Name:       "grpc-api",
						Port:       constants.AgentAPIPort,
						TargetPort: intstr.FromInt(constants.AgentAPIPort),
						Protocol:   corev1.ProtocolTCP,
					},
				},
			},
		}
		if err := controllerutil.SetControllerReference(instance, service, r.scheme); err != nil {
			return reconcile.Result{}, err
		}

		// Check if the task agent Service already exists
		found := &corev1.Service{}
		if err := r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, found); err != nil && errors.IsNotFound(err) {
			log.Info().Msg("Creating Task Agent Service")
			if err := r.Create(ctx, service); err != nil {
				log.Error().Err(err).Msg("Failed to create Task Agent Service")
				return reconcile.Result{}, err
			}
		} else if err != nil {
			return reconcile.Result{}, err
		} else {
			// Update the found object and write the result back if there are any changes
			if diff := util.ServiceEqual(*service, *found); len(diff) > 0 {
				service.Spec.ClusterIP = found.Spec.ClusterIP
				found.Spec = service.Spec
				log.Info().Interface("diff", diff).Msg("Updating Task Agent Service")
				if err := r.Update(ctx, found); err != nil {
					return reconcile.Result{}, err
				}
			}
		}
	}

	return reconcile.Result{}, nil
}
