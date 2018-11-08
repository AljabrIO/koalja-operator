// Copyright © 2018 Heptio
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

package contour

import (
	"sort"
	"sync"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/heptio/contour/internal/dag"
	"github.com/heptio/contour/internal/envoy"
)

// RouteCache manages the contents of the gRPC RDS cache.
type RouteCache struct {
	routeCache
}

type routeCache struct {
	mu      sync.Mutex
	values  map[string]*v2.RouteConfiguration
	waiters []chan int
	last    int
}

// Register registers ch to receive a value when Notify is called.
// The value of last is the count of the times Notify has been called on this Cache.
// It functions of a sequence counter, if the value of last supplied to Register
// is less than the Cache's internal counter, then the caller has missed at least
// one notification and will fire immediately.
//
// Sends by the broadcaster to ch must not block, therefor ch must have a capacity
// of at least 1.
func (c *routeCache) Register(ch chan int, last int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if last < c.last {
		// notify this channel immediately
		ch <- c.last
		return
	}
	c.waiters = append(c.waiters, ch)
}

// Update replaces the contents of the cache with the supplied map.
func (c *routeCache) Update(v map[string]*v2.RouteConfiguration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.values = v
	c.notify()
}

// notify notifies all registered waiters that an event has occurred.
func (c *routeCache) notify() {
	c.last++

	for _, ch := range c.waiters {
		ch <- c.last
	}
	c.waiters = c.waiters[:0]
}

// Values returns a slice of the value stored in the cache.
func (c *routeCache) Values(filter func(string) bool) []proto.Message {
	c.mu.Lock()
	values := make([]proto.Message, 0, len(c.values))
	for _, v := range c.values {
		if filter(v.Name) {
			values = append(values, v)
		}
	}
	c.mu.Unlock()
	return values
}

type routeVisitor struct {
	*RouteCache
	dag.Visitable
}

func (v *routeVisitor) Visit() map[string]*v2.RouteConfiguration {
	ingress_http := &v2.RouteConfiguration{
		Name: "ingress_http",
	}
	ingress_https := &v2.RouteConfiguration{
		Name: "ingress_https",
	}
	m := map[string]*v2.RouteConfiguration{
		ingress_http.Name:  ingress_http,
		ingress_https.Name: ingress_https,
	}
	v.Visitable.Visit(func(vh dag.Vertex) {
		switch vh := vh.(type) {
		case *dag.VirtualHost:
			vhost := envoy.VirtualHost(vh.Host, 80)
			vh.Visit(func(r dag.Vertex) {
				switch r := r.(type) {
				case *dag.Route:
					var svcs []*dag.HTTPService
					r.Visit(func(s dag.Vertex) {
						if s, ok := s.(*dag.HTTPService); ok {
							svcs = append(svcs, s)
						}
					})
					if len(svcs) < 1 {
						// no services for this route, skip it.
						return
					}
					rr := route.Route{
						Match:  envoy.PrefixMatch(r.Prefix),
						Action: envoy.RouteRoute(r, svcs),
					}

					if r.HTTPSUpgrade {
						rr.Action = envoy.UpgradeHTTPS()
					}
					vhost.Routes = append(vhost.Routes, rr)
				}
			})
			if len(vhost.Routes) < 1 {
				return
			}
			sort.Stable(sort.Reverse(longestRouteFirst(vhost.Routes)))
			ingress_http.VirtualHosts = append(ingress_http.VirtualHosts, vhost)
		case *dag.SecureVirtualHost:
			vhost := envoy.VirtualHost(vh.Host, 443)
			vh.Visit(func(r dag.Vertex) {
				switch r := r.(type) {
				case *dag.Route:
					var svcs []*dag.HTTPService
					r.Visit(func(s dag.Vertex) {
						if s, ok := s.(*dag.HTTPService); ok {
							svcs = append(svcs, s)
						}
					})
					if len(svcs) < 1 {
						// no services for this route, skip it.
						return
					}
					vhost.Routes = append(vhost.Routes, route.Route{
						Match:  envoy.PrefixMatch(r.Prefix),
						Action: envoy.RouteRoute(r, svcs),
					})
				}
			})
			if len(vhost.Routes) < 1 {
				return
			}
			sort.Stable(sort.Reverse(longestRouteFirst(vhost.Routes)))
			ingress_https.VirtualHosts = append(ingress_https.VirtualHosts, vhost)
		}
	})

	for _, v := range m {
		sort.Stable(virtualHostsByName(v.VirtualHosts))
	}
	return m
}

type virtualHostsByName []route.VirtualHost

func (v virtualHostsByName) Len() int           { return len(v) }
func (v virtualHostsByName) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }
func (v virtualHostsByName) Less(i, j int) bool { return v[i].Name < v[j].Name }

type longestRouteFirst []route.Route

func (l longestRouteFirst) Len() int      { return len(l) }
func (l longestRouteFirst) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l longestRouteFirst) Less(i, j int) bool {
	a, ok := l[i].Match.PathSpecifier.(*route.RouteMatch_Prefix)
	if !ok {
		// ignore non prefix matches
		return false
	}

	b, ok := l[j].Match.PathSpecifier.(*route.RouteMatch_Prefix)
	if !ok {
		// ignore non prefix matches
		return false
	}

	return a.Prefix < b.Prefix
}

func u32(val int) *types.UInt32Value { return &types.UInt32Value{Value: uint32(val)} }

var bvTrue = types.BoolValue{Value: true}

// bv returns a pointer to a true types.BoolValue if val is true,
// otherwise it returns nil.
func bv(val bool) *types.BoolValue {
	if val {
		return &bvTrue
	}
	return nil
}
