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

	"github.com/dchest/uniuri"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentsv1alpha1 "github.com/AljabrIO/koalja-operator/pkg/apis/agents/v1alpha1"
	"github.com/AljabrIO/koalja-operator/pkg/fs"
	fssvc "github.com/AljabrIO/koalja-operator/pkg/fs/service"
)

type localFSBuilder struct {
	localPathPrefix  string
	storageClassName string
	scheme           string
}

// NewLocalFileSystemBuilder creates a new builder that builds a local FS.
func NewLocalFileSystemBuilder(localPathPrefix, storageClassName, scheme string) fssvc.APIBuilder {
	return &localFSBuilder{
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
	if err := deps.Client.Get(ctx, client.ObjectKey{Name: stgClass.GetName()}, &found); err != nil {
		// Create storage class
		if err := deps.Client.Create(ctx, stgClass); err != nil {
			return nil, err
		}
	} else {
		// TODO compare StorageClass and update if needed
	}

	return &localFS{
		APIDependencies:  deps,
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
	fssvc.APIDependencies
	localPathPrefix  string
	storageClassName string
	scheme           string
}

// CreateVolumeForWrite creates a PersistentVolume that can be used to
// write files to.
func (lfs *localFS) CreateVolumeForWrite(ctx context.Context, req *fs.CreateVolumeForWriteRequest) (*fs.CreateVolumeForWriteResponse, error) {
	// Get all nodes
	var nodeList corev1.NodeList
	if err := lfs.Client.Get(ctx, client.ObjectKey{}, &nodeList); err != nil {
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
	uid := uniuri.New()
	volumePath := filepath.Join(lfs.localPathPrefix, uid)

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
			StorageClassName: lfs.storageClassName,
			NodeAffinity: &corev1.VolumeNodeAffinity{
				Required: createNodeSelector(nodeName),
			},
		},
	}
	if err := lfs.Client.Create(ctx, pv); err != nil {
		return nil, err
	}

	return &fs.CreateVolumeForWriteResponse{
		VolumeName: pv.GetName(),
		NodeName:   nodeName,
	}, nil
}

// CreateFileURI creates a URI for the given file/dir
func (lfs *localFS) CreateFileURI(ctx context.Context, req *fs.CreateFileURIRequest) (*fs.CreateFileURIResponse, error) {
	// Read original PV
	var originalPV corev1.PersistentVolume
	if err := lfs.Client.Get(ctx, client.ObjectKey{Name: req.GetVolumeName()}, &originalPV); err != nil {
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
	// Parse URI
	uri, err := url.Parse(req.GetURI())
	if err != nil {
		return nil, err
	}
	// Check scheme
	if uri.Scheme != lfs.scheme {
		return nil, fmt.Errorf("Unknown scheme '%s'", uri.Scheme)
	}
	nodeName := uri.Host
	uid := uri.Path
	localPath := uri.Fragment
	isDir, _ := strconv.ParseBool(uri.Query().Get(dirKey))

	// Create a PV
	volumePath := filepath.Join(lfs.localPathPrefix, uid)
	pv := &corev1.PersistentVolume{
		ObjectMeta: v1.ObjectMeta{
			Name: "koalja-local-ro-" + uniuri.New(),
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
				corev1.ReadOnlyMany,
			},
			StorageClassName: lfs.storageClassName,
			NodeAffinity: &corev1.VolumeNodeAffinity{
				Required: createNodeSelector(nodeName),
			},
		},
	}
	if err := lfs.Client.Create(ctx, pv); err != nil {
		return nil, err
	}

	return &fs.CreateVolumeForReadResponse{
		VolumeName: pv.GetName(),
		NodeName:   nodeName,
		LocalPath:  localPath,
		IsDir:      isDir,
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
