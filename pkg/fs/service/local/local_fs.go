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
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/AljabrIO/koalja-operator/pkg/constants"

	"github.com/dchest/uniuri"
	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentsv1alpha1 "github.com/AljabrIO/koalja-operator/pkg/apis/agents/v1alpha1"
	"github.com/AljabrIO/koalja-operator/pkg/fs"
	fssvc "github.com/AljabrIO/koalja-operator/pkg/fs/service"
	"github.com/AljabrIO/koalja-operator/pkg/util/retry"
)

type localFSBuilder struct {
	log              zerolog.Logger
	localPathPrefix  string
	storageClassName string
	scheme           string
}

// NewLocalFileSystemBuilder creates a new builder that builds a local FS.
func NewLocalFileSystemBuilder(log zerolog.Logger, localPathPrefix, storageClassName, scheme string) fssvc.APIBuilder {
	return &localFSBuilder{
		log:              log,
		localPathPrefix:  localPathPrefix,
		storageClassName: storageClassName,
		scheme:           scheme,
	}
}

// NewFileSystem builds a new local FS
func (b *localFSBuilder) NewFileSystem(deps fssvc.APIDependencies) (fs.FileSystemServer, error) {
	// Ensure storageclass
	bindingMode := storagev1.VolumeBindingWaitForFirstConsumer
	stgClass := &storagev1.StorageClass{
		ObjectMeta: v1.ObjectMeta{
			Name: b.storageClassName,
		},
		VolumeBindingMode: &bindingMode,
		Provisioner:       agentsv1alpha1.SchemeGroupVersion.Group + "/local-fs",
	}
	ctx := context.Background()
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

	return &localFS{
		APIDependencies:  deps,
		log:              b.log,
		localPathPrefix:  b.localPathPrefix,
		storageClassName: b.storageClassName,
		scheme:           b.scheme,
	}, nil
}

const (
	dirKey = "dir"
	uidKey = "koalja-uid"
)

type localFS struct {
	log zerolog.Logger
	fssvc.APIDependencies
	localPathPrefix  string
	storageClassName string
	scheme           string
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
	volumePath := filepath.Join(lfs.localPathPrefix, uid)

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
			StorageClassName:              lfs.storageClassName,
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
		Str("volName", req.GetVolumeName()).
		Str("nodeName", req.GetNodeName()).
		Str("localPath", req.GetLocalPath()).
		Bool("isDir", req.IsDir).
		Logger()
	log.Debug().Msg("CreateFileURI request")

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
		Scheme:   lfs.scheme,
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
	// Check scheme
	if uri.Scheme != lfs.scheme {
		log.Debug().Str("scheme", uri.Scheme).Msg("Unknown scheme")
		return nil, fmt.Errorf("Unknown scheme '%s'", uri.Scheme)
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
	volumePath := filepath.Join(lfs.localPathPrefix, uid)
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
			StorageClassName:              lfs.storageClassName,
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
