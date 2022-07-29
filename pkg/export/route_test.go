//go:build !integration || all
// +build !integration all

/*
 *  Copyright 2022 VMware, Inc.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *  http://www.apache.org/licenses/LICENSE-2.0
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package export

import (
	"reflect"
	"testing"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cache"
)

func TestAdjustRoutes(t *testing.T) {
	routeMapper := &routeMapper{
		DomainsToAdd: []string{"added.cf.example.com"},
		DomainsToReplace: map[string]string{
			"old.cf.example.com": "replaced.cf.example.com",
		},
	}

	appRoutes := []string{"app1.old.cf.example.com", "app1.old-ignored.cf.example.com"}
	adjustedRoutes := routeMapper.AdjustRoutes(appRoutes)

	if len(adjustedRoutes) != 3 {
		t.Fatalf("Expected 3 routes, but got %d", len(adjustedRoutes))
	}

	expectedRoutes := []string{"app1.added.cf.example.com", "app1.replaced.cf.example.com", "app1.old-ignored.cf.example.com"}

	for _, ar := range adjustedRoutes {
		found := false
		for _, er := range expectedRoutes {
			if ar == er {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("The adjusted route %s was not found in the list of expected routes", ar)
		}
	}
}

func TestRoutesWithSlashes(t *testing.T) {
	routeMapper := &routeMapper{
		DomainsToAdd: []string{"added.cf.example.com"},
		DomainsToReplace: map[string]string{
			"old.cf.example.com": "replaced.cf.example.com",
		},
	}

	appRoutes := []string{"app1.old.cf.example.com/v2/", "app1.old-ignored.cf.example.com/api/v2", "app1.old.cf.example.com", "app1.replaced.cf.example.com/v3"}
	adjustedRoutes := routeMapper.AdjustRoutes(appRoutes)

	if len(adjustedRoutes) != 5 {
		t.Fatalf("Expected 5 routes, but got %d", len(adjustedRoutes))
	}

	expectedRoutes := []string{"app1.added.cf.example.com", "app1.replaced.cf.example.com", "app1.replaced.cf.example.com/v2/", "app1.replaced.cf.example.com/v3", "app1.old-ignored.cf.example.com/api/v2"}

	for _, ar := range adjustedRoutes {
		found := false
		for _, er := range expectedRoutes {
			if ar == er {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("The adjusted route %s was not found in the list of expected routes: %v", ar, expectedRoutes)
		}
	}
}

func TestNoRoutes(t *testing.T) {
	routeMapper := &routeMapper{
		DomainsToAdd: []string{"added.cf.example.com"},
		DomainsToReplace: map[string]string{
			"old.cf.example.com": "replaced.cf.example.com",
		},
	}

	var appRoutes []string
	adjustedRoutes := routeMapper.AdjustRoutes(appRoutes)

	if len(adjustedRoutes) != 0 {
		t.Fatalf("Expected 0 routes, but got %d", len(adjustedRoutes))
	}
}

func TestNoRouteMappings(t *testing.T) {
	routeMapper := &routeMapper{}

	appRoutes := []string{"app1.old.cf.example.com", "app1.old-ignored.cf.example.com"}
	adjustedRoutes := routeMapper.AdjustRoutes(appRoutes)

	if len(adjustedRoutes) != 2 {
		t.Fatalf("Expected 2 routes, but got %d", len(adjustedRoutes))
	}

	expectedRoutes := []string{"app1.old.cf.example.com", "app1.old-ignored.cf.example.com"}

	for _, ar := range adjustedRoutes {
		found := false
		for _, er := range expectedRoutes {
			if ar == er {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("The adjusted route %s was not found in the list of expected routes", ar)
		}
	}
}

func TestAdjustRoutes_WithMultipleDomainsToReplace(t *testing.T) {
	type fields struct {
		DomainsToReplace map[string]string
	}
	type args struct {
		existingRoutes []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []string
	}{
		{
			name: "more than one domain mapping",
			fields: fields{
				DomainsToReplace: map[string]string{"foo1.com": "bar1.com", "foo2.com": "bar2.com"},
			},
			args: args{
				existingRoutes: []string{"foo1.com", "foo2.com"},
			},
			want: []string{"bar1.com", "bar2.com"},
		},
		{
			name: "more than one domain mapping to the same domain",
			fields: fields{
				DomainsToReplace: map[string]string{"foo1.com": "bar1.com", "foo2.com": "bar1.com"},
			},
			args: args{
				existingRoutes: []string{"foo1.com"},
			},
			want: []string{"bar1.com"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			r := &routeMapper{
				DomainsToReplace: tt.fields.DomainsToReplace,
			}
			if got := r.AdjustRoutes(tt.args.existingRoutes); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AdjustRoutes() = %v, want %v", got, tt.want)
			}
		})
	}
}
