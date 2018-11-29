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

package s3

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dchest/uniuri"
	"github.com/rs/zerolog"
	grpc "google.golang.org/grpc"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/AljabrIO/koalja-operator/pkg/constants"
	"github.com/AljabrIO/koalja-operator/pkg/fs"
	fssvc "github.com/AljabrIO/koalja-operator/pkg/fs/service"
	"github.com/AljabrIO/koalja-operator/pkg/util/retry"
)

const (
	myContainerName = "fs-service"
)

// Config of the S3 FS
type Config struct {
	// Name of the pod running this service
	PodName string
	// Namespace that contains the pod running this service
	Namespace string
	// MountPathPrefix is the directory on nodes where buckets are mounted from
	MountPathPrefix string
	// Image to use for the mount daemonset
	DaemonImage string
}

type s3FSBuilder struct {
	log zerolog.Logger
	Config
}

// NewS3FileSystemBuilder creates a new builder that builds an S3 FS.
func NewS3FileSystemBuilder(log zerolog.Logger, cfg Config) fssvc.APIBuilder {
	return &s3FSBuilder{
		log:    log,
		Config: cfg,
	}
}

// NewFileSystem builds a new local FS
func (b *s3FSBuilder) NewFileSystem(ctx context.Context, deps fssvc.APIDependencies) (fssvc.FileSystemServer, error) {
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

	// Fetch StorageConfig
	var storageConfig *StorageConfig
	if err := retry.Do(ctx, func(ctx context.Context) error {
		var err error
		storageConfig, err = NewStorageConfig(ctx, deps.Client, b.Config.Namespace)
		if err != nil {
			return maskAny(err)
		}
		return nil
	}, retry.Timeout(constants.TimeoutAPIServer)); err != nil {
		b.log.Error().Err(err).Msg("Failed to load Storage Config")
		return nil, maskAny(err)
	}

	return &s3FS{
		APIDependencies: deps,
		log:             b.log,
		Config:          b.Config,
		statefulSetName: ownerName,
		owner:           &sfs,
		storageConfig:   storageConfig,
	}, nil
}

const (
	queryDirKey = "dir"
	queryUIDKey = "uid"
)

type s3FS struct {
	log zerolog.Logger
	fssvc.APIDependencies
	Config
	statefulSetName string
	owner           metav1.Object
	storageConfig   *StorageConfig
}

// Register GRPC services
func (s3fs *s3FS) Register(svr *grpc.Server) {
	// Nothing needed
}

// Run until the given context is canceled
func (s3fs *s3FS) Run(ctx context.Context) error {
	log := s3fs.log
	for {
		// Ensure node daemonset is up to date
		if err := createOrUpdateNodeDaemonset(ctx, log, s3fs.Client, nodeDaemonConfig{
			Name:              s3fs.statefulSetName,
			Namespace:         s3fs.Config.Namespace,
			Image:             s3fs.Config.DaemonImage,
			NodeRootMountPath: s3fs.Config.MountPathPrefix,
			Buckets:           s3fs.storageConfig.Buckets,
		}, s3fs.owner, s3fs.APIDependencies.Scheme); err != nil {
			log.Error().Err(err).Msg("Failed to create/update node DaemonSet")
		}
	}
}

// CreateVolumeForWrite creates a PersistentVolume that can be used to
// write files to.
func (s3fs *s3FS) CreateVolumeForWrite(ctx context.Context, req *fs.CreateVolumeForWriteRequest) (*fs.CreateVolumeForWriteResponse, error) {
	log := s3fs.log.With().
		Int64("estimatedCapacity", req.GetEstimatedCapacity()).
		Str("nodeName", req.GetNodeName()).
		Logger()
	log.Debug().Msg("CreateVolumeForWrite request")

	// Now create a unique local path of the node
	uid := strings.ToLower(uniuri.New())
	bucketMountPath := s3fs.storageConfig.Buckets[0].NodeMountPoint(s3fs.Config.MountPathPrefix)
	volumePath := filepath.Join(bucketMountPath, uid)

	return &fs.CreateVolumeForWriteResponse{
		VolumePath:     volumePath,
		DeleteAfterUse: true,
	}, nil
}

// CreateFileURI creates a URI for the given file/dir
func (s3fs *s3FS) CreateFileURI(ctx context.Context, req *fs.CreateFileURIRequest) (*fs.CreateFileURIResponse, error) {
	log := s3fs.log.With().
		Str("scheme", req.GetScheme()).
		Str("volName", req.GetVolumeName()).
		Str("volPath", req.GetVolumePath()).
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

	// Find UID
	var uid, mountPath string
	if req.GetVolumePath() != "" {
		// Extract UID from volume path
		uid = path.Base(req.GetVolumePath())
		mountPath = path.Dir(req.GetVolumePath())
	} else {
		// Invalid request
		return nil, fmt.Errorf("VolumePath cannot be empty")
	}

	// Find bucket for mountpath
	var bucket BucketConfig
	foundBucket := false
	for _, bc := range s3fs.storageConfig.Buckets {
		if bc.NodeMountPoint(s3fs.Config.MountPathPrefix) == mountPath {
			bucket = bc
			foundBucket = true
			break
		}
	}
	if !foundBucket {
		return nil, fmt.Errorf("Bucket cannot be found for given VolumePath")
	}

	// Create URI
	q := url.Values{}
	q.Set(queryDirKey, strconv.FormatBool(req.GetIsDir()))
	q.Set(queryUIDKey, uid)
	uri := &url.URL{
		Scheme:   req.GetScheme(),
		Host:     bucket.Endpoint,
		Path:     bucket.Name,
		Fragment: req.GetLocalPath(),
		RawQuery: q.Encode(),
	}
	return &fs.CreateFileURIResponse{
		URI: uri.String(),
	}, nil
}

// CreateVolumeForRead creates a PersistentVolume for reading a given URI
func (s3fs *s3FS) CreateVolumeForRead(ctx context.Context, req *fs.CreateVolumeForReadRequest) (*fs.CreateVolumeForReadResponse, error) {
	log := s3fs.log.With().
		Str("uri", req.GetURI()).
		Logger()
	log.Debug().Msg("CreateVolumeForRead request")

	// Parse URI
	uri, err := url.Parse(req.GetURI())
	if err != nil {
		log.Debug().Err(err).Msg("Failed to parse URI")
		return nil, err
	}
	q := uri.Query()
	endpoint := uri.Host
	bucketName := strings.TrimPrefix(uri.Path, "/")
	localPath := uri.Fragment
	uid := q.Get(queryUIDKey)
	isDir, _ := strconv.ParseBool(q.Get(queryDirKey))

	// Find bucket for endpoint & name
	var bucket BucketConfig
	foundBucket := false
	for _, bc := range s3fs.storageConfig.Buckets {
		if bc.Matches(endpoint, bucketName) {
			bucket = bc
			foundBucket = true
			break
		}
	}
	if !foundBucket {
		return nil, fmt.Errorf("Bucket cannot be found for given endpoint & name")
	}

	// Create volume path
	volumePath := filepath.Join(bucket.NodeMountPoint(s3fs.Config.MountPathPrefix), uid)

	// Return response
	return &fs.CreateVolumeForReadResponse{
		VolumePath:     volumePath,
		LocalPath:      localPath,
		IsDir:          isDir,
		DeleteAfterUse: true,
	}, nil
}

// CreateFileView returns a view on the given file identified by the given URI.
func (s3fs *s3FS) CreateFileView(ctx context.Context, req *fs.CreateFileViewRequest) (*fs.CreateFileViewResponse, error) {
	log := s3fs.log.With().
		Str("uri", req.GetURI()).
		Logger()
	log.Debug().Msg("CreateFileView request")

	// Parse URI
	uri, err := url.Parse(req.GetURI())
	if err != nil {
		log.Debug().Err(err).Msg("Failed to parse URI")
		return nil, err
	}
	q := uri.Query()
	endpoint := uri.Host
	bucketName := strings.TrimPrefix(uri.Path, "/")
	localPath := uri.Fragment
	uid := q.Get(queryUIDKey)
	//isDir, _ := strconv.ParseBool(q.Get(queryDirKey))

	// Find bucket for endpoint & name
	var bucket BucketConfig
	foundBucket := false
	for _, bc := range s3fs.storageConfig.Buckets {
		if bc.Matches(endpoint, bucketName) {
			bucket = bc
			foundBucket = true
			break
		}
	}
	if !foundBucket {
		return nil, fmt.Errorf("Bucket cannot be found for given endpoint & name")
	}

	// Create volume path
	volumePath := filepath.Join(bucket.NodeMountPoint(s3fs.Config.MountPathPrefix), uid)
	fullPath := filepath.Join(volumePath, localPath)

	// Read file
	content, err := ioutil.ReadFile(fullPath)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to read file")
		return nil, err
	}

	// Detect content type
	contentType := http.DetectContentType(content)

	return &fs.CreateFileViewResponse{
		Content:     content,
		ContentType: contentType,
	}, nil
}
