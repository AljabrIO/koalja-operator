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

package client

import (
	"context"
	"fmt"
	"os"

	"github.com/AljabrIO/koalja-operator/pkg/constants"
	"github.com/AljabrIO/koalja-operator/pkg/fs"
	"google.golang.org/grpc"
)

// FileSystemClient is a closable client interface.
type FileSystemClient interface {
	fs.FileSystemClient
	Close() error
}

// NewFileSystemClient creates a client for the filesystem service, which
// address is found in the environment
func NewFileSystemClient() (FileSystemClient, error) {
	address := os.Getenv(constants.EnvFileSystemAddress)
	if address == "" {
		return nil, fmt.Errorf("Environment variable '%s' not set", constants.EnvFileSystemAddress)
	}

	// Create a connection
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	// Create a client
	c := fs.NewFileSystemClient(conn)

	return &fileSystemClient{c: c, conn: conn}, nil
}

type fileSystemClient struct {
	c    fs.FileSystemClient
	conn *grpc.ClientConn
}

// CreateVolumeForWrite creates a PersistentVolume that can be used to
// write files to.
func (c *fileSystemClient) CreateVolumeForWrite(ctx context.Context, in *fs.CreateVolumeForWriteRequest, opts ...grpc.CallOption) (*fs.CreateVolumeForWriteResponse, error) {
	return c.c.CreateVolumeForWrite(ctx, in, opts...)
}

// CreateFileURI creates a URI for the given file/dir
func (c *fileSystemClient) CreateFileURI(ctx context.Context, in *fs.CreateFileURIRequest, opts ...grpc.CallOption) (*fs.CreateFileURIResponse, error) {
	return c.c.CreateFileURI(ctx, in, opts...)
}

// CreateVolumeForRead creates a PersistentVolume for reading a given URI
func (c *fileSystemClient) CreateVolumeForRead(ctx context.Context, in *fs.CreateVolumeForReadRequest, opts ...grpc.CallOption) (*fs.CreateVolumeForReadResponse, error) {
	return c.c.CreateVolumeForRead(ctx, in, opts...)
}

// CreateFileView returns a view on the given file identified by the given URI.
func (c *fileSystemClient) CreateFileView(ctx context.Context, in *fs.CreateFileViewRequest, opts ...grpc.CallOption) (*fs.CreateFileViewResponse, error) {
	return c.c.CreateFileView(ctx, in, opts...)
}

// Close the connection
func (c *fileSystemClient) Close() error {
	return c.conn.Close()
}
