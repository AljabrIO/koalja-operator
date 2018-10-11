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

package v1alpha1

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

var (
	namePattern = regexp.MustCompile("[a-zA-Z][a-zA-Z_0-9]*")
)

// ValidateName checks the given name for being a valid name of task, link, type etc.
func ValidateName(name string) error {
	if name == "" {
		return errors.Wrapf(ErrValidation, "Name expected")
	}
	if !namePattern.MatchString(name) {
		return errors.Wrapf(ErrValidation, "Invalid name '%s'", name)
	}
	return nil
}

// SplitTaskRef splits a task reference <name>/<i/o-name>
func SplitTaskRef(ref string) (taskName, ioName string, err error) {
	parts := strings.Split(ref, "/")
	if len(parts) != 2 {
		return "", "", errors.Wrapf(ErrValidation, "Expected '<name>/<i/o-name>', got '%s'", ref)
	}
	if len(parts[0]) == 0 {
		return "", "", errors.Wrapf(ErrValidation, "Name expected, got '%s'", ref)
	}
	if len(parts[1]) == 0 {
		return "", "", errors.Wrapf(ErrValidation, "Input/output name expected, got '%s'", ref)
	}
	return parts[0], parts[1], nil
}
