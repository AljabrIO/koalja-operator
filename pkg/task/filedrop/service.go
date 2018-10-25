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

package filedrop

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/AljabrIO/koalja-operator/pkg/constants"
	fs "github.com/AljabrIO/koalja-operator/pkg/fs/client"
	taskclient "github.com/AljabrIO/koalja-operator/pkg/task/client"
	"github.com/AljabrIO/koalja-operator/pkg/util"
)

// Config holds the configuration arguments of the service.
type Config struct {
	// Local directory path where to drop files
	DropFolder string
	// Name of volume that contains DropFolder
	VolumeName string
	// Mount path of volume that contains DropFolder
	MountPath string
	// Name of node we're running on
	NodeName string
	// Name of the task output we're serving
	OutputName string
}

// Service loop of the FileDrop task.
type Service struct {
	Config
	log       zerolog.Logger
	k8sClient client.Client
	fsClient  fs.FileSystemClient
	ornClient taskclient.OutputReadyNotifierClient
	pod       *corev1.Pod
	scheme    *runtime.Scheme
}

// NewService initializes a new service.
func NewService(cfg Config, log zerolog.Logger, config *rest.Config, scheme *runtime.Scheme) (*Service, error) {
	// Check arguments
	if cfg.DropFolder == "" {
		return nil, fmt.Errorf("DropFolder expected")
	}
	if cfg.VolumeName == "" {
		return nil, fmt.Errorf("VolumeName expected")
	}
	if cfg.MountPath == "" {
		return nil, fmt.Errorf("MountPath expected")
	}
	if cfg.NodeName == "" {
		return nil, fmt.Errorf("NodeName expected")
	}
	if cfg.OutputName == "" {
		return nil, fmt.Errorf("OutputName expected")
	}

	// Create k8s client
	var k8sClient client.Client
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := util.Retry(ctx, func(ctx context.Context) error {
		var err error
		k8sClient, err = client.New(config, client.Options{Scheme: scheme})
		return err
	}); err != nil {
		log.Error().Err(err).Msg("Failed to create k8s client")
		return nil, maskAny(err)
	}

	// Load my own pod
	var pod corev1.Pod
	podKey := client.ObjectKey{
		Name:      os.Getenv(constants.EnvPodName),
		Namespace: os.Getenv(constants.EnvNamespace),
	}
	if err := util.Retry(ctx, func(ctx context.Context) error {
		return k8sClient.Get(ctx, podKey, &pod)
	}); err != nil {
		log.Error().Err(err).Msg("Failed to get my own pod")
		return nil, maskAny(err)
	}

	// Create service clients
	fsClient, err := fs.CreateFileSystemClient()
	if err != nil {
		log.Error().Err(err).Msg("Failed to create filesystem client")
		return nil, maskAny(err)
	}
	ornClient, err := taskclient.CreateOutputReadyNotifierClient()
	if err != nil {
		log.Error().Err(err).Msg("Failed to create output ready notifier client")
		return nil, maskAny(err)
	}
	return &Service{
		Config:    cfg,
		log:       log,
		k8sClient: k8sClient,
		fsClient:  fsClient,
		ornClient: ornClient,
		pod:       &pod,
		scheme:    scheme,
	}, nil
}

// Run th service until the given context is canceled
func (s *Service) Run(ctx context.Context) error {
	// Create service to access myself
	if err := s.createService(ctx, s.pod); err != nil {
		return maskAny(err)
	}
	// Run webserver
	if err := s.runServer(ctx); err != nil {
		return maskAny(err)
	}
	return nil
}

// createService creates a service that selects my pod.
func (s *Service) createService(ctx context.Context, pod *corev1.Pod) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.GetName(),
			Namespace: pod.GetNamespace(),
		},
		Spec: corev1.ServiceSpec{
			Selector: pod.GetLabels(),
			Type:     corev1.ServiceTypeNodePort,
			Ports: []corev1.ServicePort{
				corev1.ServicePort{
					Name:       "http-upload",
					Port:       8080,
					TargetPort: intstr.FromInt(8080),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
	if err := controllerutil.SetControllerReference(pod, service, s.scheme); err != nil {
		return maskAny(err)
	}
	if err := s.k8sClient.Create(ctx, service); err != nil {
		s.log.Error().Err(err).Msg("Failed to create k8s Service")
		return maskAny(err)
	}

	return nil
}
