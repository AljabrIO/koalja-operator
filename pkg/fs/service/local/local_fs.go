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

package local

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dchest/uniuri"
	"github.com/rs/zerolog"
	grpc "google.golang.org/grpc"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentsv1alpha1 "github.com/AljabrIO/koalja-operator/pkg/apis/agents/v1alpha1"
	"github.com/AljabrIO/koalja-operator/pkg/constants"
	"github.com/AljabrIO/koalja-operator/pkg/fs"
	fssvc "github.com/AljabrIO/koalja-operator/pkg/fs/service"
	"github.com/AljabrIO/koalja-operator/pkg/util"
	"github.com/AljabrIO/koalja-operator/pkg/util/retry"
)

const (
	myContainerName = "fs-service"
)

// Config of the Local FS
type Config struct {
	// Name of the pod running this service
	PodName string
	// Namespace that contains the pod running this service
	Namespace string
	// LocalPathPrefix is the directory on nodes where volumes are created from
	LocalPathPrefix string
	// Name of the StorageClass used for volumes created here
	StorageClassName string
}

type localFSBuilder struct {
	log zerolog.Logger
	Config
}

// NewLocalFileSystemBuilder creates a new builder that builds a local FS.
func NewLocalFileSystemBuilder(log zerolog.Logger, cfg Config) fssvc.APIBuilder {
	return &localFSBuilder{
		log:    log,
		Config: cfg,
	}
}

// NewFileSystem builds a new local FS
func (b *localFSBuilder) NewFileSystem(ctx context.Context, deps fssvc.APIDependencies) (fssvc.FileSystemServer, error) {
	// Fetch my pod
	var pod corev1.Pod
	if err := retry.Do(ctx, func(ctx context.Context) error {
		if err := deps.Client.Get(ctx, client.ObjectKey{Name: b.Config.PodName, Namespace: b.Config.Namespace}, &pod); err != nil {
			return maskAny(err)
		}
		return nil
	}, retry.Timeout(constants.TimeoutAPIServer)); err != nil {
		b.log.Error().Err(err).Msg("Failed to get my own Pod")
		return nil, maskAny(err)
	}
	// Find container image from image ID in container status
	var image string
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Name == myContainerName {
			image = util.ConvertImageID2Image(cs.ImageID)
		}
	}
	if image == "" {
		// Try lookup image from container spec
		for _, c := range pod.Spec.Containers {
			if c.Name == myContainerName {
				image = c.Image
			}
		}
	}
	if image == "" {
		b.log.Debug().Msg("No image found in my pod")
		return nil, maskAny(fmt.Errorf("No image found in my pod"))
	}

	// Find owner of pod
	var ownerName string
	for _, owner := range pod.GetOwnerReferences() {
		if owner.Kind == "StatefulSet" {
			ownerName = owner.Name
		}
	}
	if ownerName == "" {
		b.log.Debug().Msg("No StatefulSet owner found in my pod")
		return nil, maskAny(fmt.Errorf("No StatefulSet owner found in my pod"))
	}

	// Fetch StatefulSet
	var sfs appsv1.StatefulSet
	if err := retry.Do(ctx, func(ctx context.Context) error {
		if err := deps.Client.Get(ctx, client.ObjectKey{Name: ownerName, Namespace: b.Config.Namespace}, &sfs); err != nil {
			return maskAny(err)
		}
		return nil
	}, retry.Timeout(constants.TimeoutAPIServer)); err != nil {
		b.log.Error().Err(err).Msg("Failed to get StatefulSet that owns my own Pod")
		return nil, maskAny(err)
	}
	port, err := constants.GetAPIPort()
	if err != nil {
		b.log.Debug().Err(err).Msg("Cannot get API port")
		return nil, err
	}
	nodeRegistryAddress := net.JoinHostPort(sfs.Spec.ServiceName, strconv.Itoa(port))

	// Ensure storageclass
	bindingMode := storagev1.VolumeBindingWaitForFirstConsumer
	stgClass := &storagev1.StorageClass{
		ObjectMeta: v1.ObjectMeta{
			Name: b.StorageClassName,
		},
		VolumeBindingMode: &bindingMode,
		Provisioner:       agentsv1alpha1.SchemeGroupVersion.Group + "/local-fs",
	}
	var found storagev1.StorageClass
	if err := retry.Do(ctx, func(ctx context.Context) error {
		if err := deps.Client.Get(ctx, client.ObjectKey{Name: stgClass.GetName()}, &found); err != nil {
			// Create storage class
			if err := deps.Client.Create(ctx, stgClass); err != nil {
				return err
			}
		} else {
			// TODO compare StorageClass and update if needed
		}
		return nil
	}, retry.Timeout(constants.TimeoutAPIServer)); err != nil {
		b.log.Error().Err(err).Msg("Failed to get/create StorageClass")
		return nil, err
	}

	nodeRegistry := newNodeRegistry(b.log)

	return &localFS{
		APIDependencies:     deps,
		log:                 b.log,
		Config:              b.Config,
		nodeRegistry:        nodeRegistry,
		nodeRegistryAddress: nodeRegistryAddress,
		statefulSetName:     ownerName,
		image:               image,
		owner:               &sfs,
	}, nil
}

const (
	dirKey = "dir"
	uidKey = "koalja-uid"
)

type localFS struct {
	log zerolog.Logger
	fssvc.APIDependencies
	Config
	nodeRegistry        *nodeRegistry
	nodeRegistryAddress string
	statefulSetName     string
	image               string
	owner               metav1.Object
}

// Register GRPC services
func (lfs *localFS) Register(svr *grpc.Server) {
	RegisterNodeRegistryServer(svr, lfs.nodeRegistry)
}

// Run until the given context is canceled
func (lfs *localFS) Run(ctx context.Context) error {
	log := lfs.log
	for {
		// Ensure node daemonset is up to date
		if err := createOrUpdateNodeDaemonset(ctx, log, lfs.Client, nodeDaemonConfig{
			Name:            lfs.statefulSetName,
			Namespace:       lfs.Config.Namespace,
			Image:           lfs.image,
			RegistryAddress: lfs.nodeRegistryAddress,
			NodeVolumePath:  lfs.Config.LocalPathPrefix,
		}, lfs.owner, lfs.APIDependencies.Scheme); err != nil {
			log.Error().Err(err).Msg("Failed to create/update node DaemonSet")
		}
	}
}

// CreateVolumeForWrite creates a PersistentVolume that can be used to
// write files to.
func (lfs *localFS) CreateVolumeForWrite(ctx context.Context, req *fs.CreateVolumeForWriteRequest) (*fs.CreateVolumeForWriteResponse, error) {
	log := lfs.log.With().
		Int64("estimatedCapacity", req.GetEstimatedCapacity()).
		Str("nodeName", req.GetNodeName()).
		Logger()
	log.Debug().Msg("CreateVolumeForWrite request")

	// Get all nodes
	var nodeList corev1.NodeList
	if err := lfs.Client.List(ctx, &client.ListOptions{}, &nodeList); err != nil {
		log.Warn().Err(err).Msg("Failed to list nodes")
		return nil, err
	}
	// If not set, it must exist
	nodeName := req.GetNodeName()
	if nodeName != "" {
		found := false
		for _, n := range nodeList.Items {
			if n.GetName() == nodeName {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("Node '%s' not found", nodeName)
		}
	} else {
		// Pick a random node name
		nodeName = nodeList.Items[rand.Intn(len(nodeList.Items))].GetName()
	}

	// Now create a unique local path of the node
	uid := strings.ToLower(uniuri.New())
	volumePath := filepath.Join(lfs.LocalPathPrefix, uid)

	// Prepare storage quantity
	q, err := resource.ParseQuantity("8Gi")
	if err != nil {
		return nil, err
	}

	// Create a PV
	pv := &corev1.PersistentVolume{
		ObjectMeta: v1.ObjectMeta{
			Name: "koalja-local-" + uid,
			Labels: map[string]string{
				uidKey: uid,
			},
		},
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: volumePath,
				},
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: q,
			},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			StorageClassName:              lfs.StorageClassName,
			NodeAffinity: &corev1.VolumeNodeAffinity{
				Required: createNodeSelector(nodeName),
			},
		},
	}
	if owner := req.GetOwner(); owner != nil {
		pv.ObjectMeta.SetOwnerReferences(append(pv.ObjectMeta.GetOwnerReferences(), *owner))
	}
	if err := lfs.Client.Create(ctx, pv); err != nil {
		log.Warn().Err(err).Msg("Failed to create PersistentVolume")
		return nil, err
	}

	return &fs.CreateVolumeForWriteResponse{
		VolumeName:     pv.GetName(),
		NodeName:       nodeName,
		DeleteAfterUse: true,
	}, nil
}

// CreateFileURI creates a URI for the given file/dir
func (lfs *localFS) CreateFileURI(ctx context.Context, req *fs.CreateFileURIRequest) (*fs.CreateFileURIResponse, error) {
	log := lfs.log.With().
		Str("scheme", req.GetScheme()).
		Str("volName", req.GetVolumeName()).
		Str("nodeName", req.GetNodeName()).
		Str("localPath", req.GetLocalPath()).
		Bool("isDir", req.IsDir).
		Logger()
	log.Debug().Msg("CreateFileURI request")

	// Check arguments
	if req.GetScheme() == "" {
		return nil, fmt.Errorf("Scheme cannot be empty")
	}
	if req.GetLocalPath() == "" {
		return nil, fmt.Errorf("LocalPath cannot be empty")
	}
	if req.GetNodeName() == "" {
		return nil, fmt.Errorf("NodeName cannot be empty")
	}

	// Read original PV
	var originalPV corev1.PersistentVolume
	if err := lfs.Client.Get(ctx, client.ObjectKey{Name: req.GetVolumeName()}, &originalPV); err != nil {
		log.Warn().Msg("Failed to get PersistentVolume")
		return nil, err
	}
	// Fetch UID
	uid := originalPV.GetLabels()[uidKey]

	// Create URI
	q := url.Values{}
	q.Set(dirKey, strconv.FormatBool(req.GetIsDir()))
	uri := &url.URL{
		Scheme:   req.GetScheme(),
		Host:     req.GetNodeName(),
		Path:     uid,
		Fragment: req.GetLocalPath(),
		RawQuery: q.Encode(),
	}
	return &fs.CreateFileURIResponse{
		URI: uri.String(),
	}, nil
}

// CreateVolumeForRead creates a PersistentVolume for reading a given URI
func (lfs *localFS) CreateVolumeForRead(ctx context.Context, req *fs.CreateVolumeForReadRequest) (*fs.CreateVolumeForReadResponse, error) {
	log := lfs.log.With().
		Str("uri", req.GetURI()).
		Logger()
	log.Debug().Msg("CreateVolumeForRead request")

	// Parse URI
	uri, err := url.Parse(req.GetURI())
	if err != nil {
		log.Debug().Err(err).Msg("Failed to parse URI")
		return nil, err
	}
	nodeName := uri.Host
	uid := strings.TrimPrefix(uri.Path, "/")
	localPath := uri.Fragment
	isDir, _ := strconv.ParseBool(uri.Query().Get(dirKey))

	// Prepare storage quantity
	q, err := resource.ParseQuantity("8Gi")
	if err != nil {
		return nil, err
	}

	// Create a PV
	volumePath := filepath.Join(lfs.LocalPathPrefix, uid)
	pv := &corev1.PersistentVolume{
		ObjectMeta: v1.ObjectMeta{
			Name: "koalja-local-ro-" + strings.ToLower(uniuri.New()),
			Labels: map[string]string{
				uidKey: uid,
			},
		},
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: volumePath,
				},
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: q,
			},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			StorageClassName:              lfs.StorageClassName,
			NodeAffinity: &corev1.VolumeNodeAffinity{
				Required: createNodeSelector(nodeName),
			},
		},
	}
	if owner := req.GetOwner(); owner != nil {
		pv.ObjectMeta.SetOwnerReferences(append(pv.ObjectMeta.GetOwnerReferences(), *owner))
	}
	if err := lfs.Client.Create(ctx, pv); err != nil {
		log.Warn().Err(err).Msg("Failed to create PersistentVolume")
		return nil, err
	}

	return &fs.CreateVolumeForReadResponse{
		VolumeName:     pv.GetName(),
		NodeName:       nodeName,
		LocalPath:      localPath,
		IsDir:          isDir,
		DeleteAfterUse: true,
	}, nil
}

// CreateFileView returns a view on the given file identified by the given URI.
func (lfs *localFS) CreateFileView(ctx context.Context, req *fs.CreateFileViewRequest) (*fs.CreateFileViewResponse, error) {
	log := lfs.log.With().
		Str("uri", req.GetURI()).
		Logger()
	log.Debug().Msg("CreateFileView request")

	// Parse URI
	uri, err := url.Parse(req.GetURI())
	if err != nil {
		log.Debug().Err(err).Msg("Failed to parse URI")
		return nil, err
	}
	nodeName := uri.Host
	uid := strings.TrimPrefix(uri.Path, "/")
	localPath := uri.Fragment
	//isDir, _ := strconv.ParseBool(uri.Query().Get(dirKey))
	volumePath := filepath.Join(lfs.LocalPathPrefix, uid)
	fullPath := filepath.Join(volumePath, localPath)

	// Find node client
	c, err := lfs.nodeRegistry.GetNodeClient(ctx, nodeName)
	if err != nil {
		log.Debug().Err(err).Str("node", nodeName).Msg("Failed to get/create node client")
		return nil, err
	} else if c == nil {
		log.Debug().Err(err).Str("node", nodeName).Msg("No node was registered")
		return nil, fmt.Errorf("Unknown node '%s'", nodeName)
	}

	// Pass call through
	resp, err := c.CreateFileView(ctx, &CreateFileViewRequest{
		LocalPath: fullPath,
		Preview:   req.GetPreview(),
	})
	if err != nil {
		log.Debug().Err(err).Str("node", nodeName).Msg("CreateFileView failed")
		return nil, err
	}
	return resp, nil
}

// createNodeAffinity creates a node affinity for the given node name.
func createNodeSelector(nodeName string) *corev1.NodeSelector {
	return &corev1.NodeSelector{
		NodeSelectorTerms: []corev1.NodeSelectorTerm{
			corev1.NodeSelectorTerm{
				MatchExpressions: []corev1.NodeSelectorRequirement{
					corev1.NodeSelectorRequirement{
						Key:      "kubernetes.io/hostname",
						Operator: corev1.NodeSelectorOpIn,
						Values:   []string{nodeName},
					},
				},
			},
		},
	}
}
