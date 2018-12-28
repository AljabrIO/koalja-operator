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

package arangodb

import (
	driver "github.com/arangodb/go-driver"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	maskAny = errors.WithStack
)

// arangoErrorToGRPC converts an ArangoDB error to a GRPC error.
func arangoErrorToGRPC(err error) error {
	if err == nil {
		return nil
	} else if driver.IsNotFound(err) {
		return status.Error(codes.NotFound, err.Error())
	} else if driver.IsConflict(err) {
		return status.Error(codes.AlreadyExists, err.Error())
	} else if driver.IsPreconditionFailed(err) {
		return status.Error(codes.FailedPrecondition, err.Error())
	} else if driver.IsUnauthorized(err) {
		return status.Error(codes.Unauthenticated, err.Error())
	} else {
		return status.Error(codes.Unknown, err.Error())
	}
}
