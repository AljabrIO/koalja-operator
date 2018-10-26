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

package retry

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/onsi/gomega"
)

func TestRetry(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	g.Expect(Do(nil, func(context.Context) error { return nil })).To(gomega.Succeed())
	g.Expect(Do(nil, func(context.Context) error { return Permanent(fmt.Errorf("foo")) })).ToNot(gomega.Succeed())
	ctx := context.Background()
	g.Expect(Do(ctx, func(context.Context) error { return nil })).To(gomega.Succeed())
	g.Expect(Do(ctx, func(context.Context) error { return Permanent(fmt.Errorf("foo")) })).ToNot(gomega.Succeed())
}

func TestRetryTimeout1(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	called := 0
	g.Expect(Do(ctx, func(ctx context.Context) error { called++; <-ctx.Done(); return nil })).ToNot(gomega.Succeed())
	g.Expect(called).To(gomega.And(gomega.BeNumerically(">=", 1), gomega.BeNumerically("<=", 3)))
}

func TestRetryTimeoutOpt(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	ctx := context.Background()
	called := 0
	g.Expect(Do(ctx, func(ctx context.Context) error { called++; <-ctx.Done(); return nil }, Timeout(time.Millisecond*10), MinAttempts(1))).ToNot(gomega.Succeed())
	g.Expect(called).To(gomega.Equal(1))
	called = 0
	g.Expect(Do(ctx, func(ctx context.Context) error { called++; <-ctx.Done(); return nil }, Timeout(time.Millisecond*10), MinAttempts(2))).ToNot(gomega.Succeed())
	g.Expect(called).To(gomega.Equal(2))
}

func TestRetryTimeout3(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
	defer cancel()
	called := 0
	g.Expect(Do(ctx, func(ctx context.Context) error { called++; <-ctx.Done(); return nil }, MinAttempts(3))).ToNot(gomega.Succeed())
	g.Expect(called).To(gomega.Equal(3))
}
