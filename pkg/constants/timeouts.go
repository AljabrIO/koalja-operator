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

package constants

import (
	"time"
)

const (
	// TimeoutK8sClient is the timeout of creating a k8s client.
	// This is a bit long since it is often done early in the lifetime of a process.
	TimeoutK8sClient = time.Second * 30
	// TimeoutAPIServer is the timeout of a API server call. (assuming 3 attempts)
	TimeoutAPIServer = time.Second * 15
	// TimeoutRegisterAgent is the timeout of a register agent call.
	TimeoutRegisterAgent = time.Minute
)
