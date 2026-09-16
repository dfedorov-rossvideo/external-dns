/*
Copyright 2023 The Kubernetes Authors.

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

package source

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	discoveryfake "k8s.io/client-go/discovery/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
	clientgotesting "k8s.io/client-go/testing"
	v1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/gateway-api/apis/v1alpha2"
)

func setFakeGatewayRouteDiscoveryResources(kubeClient *kubefake.Clientset, resourceName string, groupVersions ...metav1.GroupVersion) {
	apiResourceLists := make([]*metav1.APIResourceList, 0, len(groupVersions))
	for _, groupVersion := range groupVersions {
		apiResourceLists = append(apiResourceLists, &metav1.APIResourceList{
			GroupVersion: groupVersion.String(),
			APIResources: []metav1.APIResource{{Name: resourceName}},
		})
	}
	kubeClient.Discovery().(*discoveryfake.FakeDiscovery).Resources = apiResourceLists
}

func TestFindServedGatewayRouteGroupVersionPrefersV1(t *testing.T) {
	kubeClient := kubefake.NewClientset()
	setFakeGatewayRouteDiscoveryResources(kubeClient, "tcproutes", v1.GroupVersion, v1alpha2.GroupVersion)

	groupVersion, err := findServedGatewayRouteGroupVersion(kubeClient.Discovery(), "tcproutes")

	require.NoError(t, err)
	require.Equal(t, v1.GroupVersion, groupVersion)
}

func TestFindServedGatewayRouteGroupVersionFallsBackToV1alpha2(t *testing.T) {
	kubeClient := kubefake.NewClientset()
	setFakeGatewayRouteDiscoveryResources(kubeClient, "tcproutes", v1alpha2.GroupVersion)

	groupVersion, err := findServedGatewayRouteGroupVersion(kubeClient.Discovery(), "tcproutes")

	require.NoError(t, err)
	require.Equal(t, v1alpha2.GroupVersion, groupVersion)
}

func TestFindServedGatewayRouteGroupVersionRejectsMissingResource(t *testing.T) {
	kubeClient := kubefake.NewClientset()
	setFakeGatewayRouteDiscoveryResources(kubeClient, "httproutes", v1.GroupVersion, v1alpha2.GroupVersion)

	_, err := findServedGatewayRouteGroupVersion(kubeClient.Discovery(), "tcproutes")

	require.EqualError(t, err, `gateway API resource "tcproutes" is not served at gateway.networking.k8s.io/v1 or gateway.networking.k8s.io/v1alpha2`)
}

func TestFindServedGatewayRouteGroupVersionReturnsDiscoveryError(t *testing.T) {
	kubeClient := kubefake.NewClientset()
	fakeDiscovery := kubeClient.Discovery().(*discoveryfake.FakeDiscovery)
	fakeDiscovery.PrependReactor("get", "resource", func(clientgotesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("discovery unavailable")
	})

	_, err := findServedGatewayRouteGroupVersion(kubeClient.Discovery(), "tcproutes")

	require.EqualError(t, err, "discover Gateway API resources at gateway.networking.k8s.io/v1: discovery unavailable")
}

func TestGatewayMatchingHost(t *testing.T) {
	tests := []struct {
		desc string
		a, b string
		host string
		ok   bool
	}{
		{
			desc: "ipv4-rejected",
			a:    "1.2.3.4",
			ok:   false,
		},
		{
			desc: "ipv6-rejected",
			a:    "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			ok:   false,
		},
		{
			desc: "empty-matches-empty",
			ok:   true,
		},
		{
			desc: "empty-matches-nonempty",
			a:    "example.net",
			host: "example.net",
			ok:   true,
		},
		{
			desc: "simple-match",
			a:    "example.net",
			b:    "example.net",
			host: "example.net",
			ok:   true,
		},
		{
			desc: "wildcard-matches-longer",
			a:    "*.example.net",
			b:    "test.example.net",
			host: "test.example.net",
			ok:   true,
		},
		{
			desc: "wildcard-matches-equal-length",
			a:    "*.example.net",
			b:    "a.example.net",
			host: "a.example.net",
			ok:   true,
		},
		{
			desc: "wildcard-matches-multiple-subdomains",
			a:    "*.example.net",
			b:    "foo.bar.test.example.net",
			host: "foo.bar.test.example.net",
			ok:   true,
		},
		{
			desc: "wildcard-doesnt-match-parent",
			a:    "*.example.net",
			b:    "example.net",
			ok:   false,
		},
		{
			desc: "wildcard-must-be-complete-label",
			a:    "*example.net",
			b:    "test.example.net",
			ok:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			for range 2 {
				if host, ok := gwMatchingHost(tt.a, tt.b); host != tt.host || ok != tt.ok {
					t.Errorf(
						"gwMatchingHost(%q, %q); got: %q, %v; want: %q, %v",
						tt.a, tt.b, host, ok, tt.host, tt.ok,
					)
				}
				tt.a, tt.b = tt.b, tt.a
			}
		})

	}
}

func TestGatewayMatchingProtocol(t *testing.T) {
	tests := []struct {
		route, lis string
		desc       string
		ok         bool
	}{
		{
			desc:  "protocol-matches-lis-https-route-http",
			route: "HTTP",
			lis:   "HTTPS",
			ok:    true,
		},
		{
			desc:  "protocol-match-invalid-list-https-route-tcp",
			route: "TCP",
			lis:   "HTTPS",
			ok:    false,
		},
		{
			desc:  "protocol-match-valid-lis-tls-route-tls",
			route: "TLS",
			lis:   "TLS",
			ok:    true,
		},
		{
			desc:  "protocol-match-valid-lis-TLS-route-TCP",
			route: "TCP",
			lis:   "TLS",
			ok:    true,
		},
		{
			desc:  "protocol-match-valid-lis-TLS-route-TCP",
			route: "TLS",
			lis:   "TCP",
			ok:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			for range 2 {
				if ok := gwProtocolMatches(v1.ProtocolType(tt.route), v1.ProtocolType(tt.lis)); ok != tt.ok {
					t.Errorf(
						"gwProtocolMatches(%q, %q); got: %v; want: %v",
						tt.route, tt.lis, ok, tt.ok,
					)
				}
				// tt.a, tt.b = tt.b, tt.a
			}
		})

	}
}

func TestIsDNS1123Domain(t *testing.T) {
	tests := []struct {
		desc string
		in   string
		ok   bool
	}{
		{
			desc: "empty",
			ok:   false,
		},
		{
			desc: "label-too-long",
			in:   strings.Repeat("x", 64) + ".example.net",
			ok:   false,
		},
		{
			desc: "domain-too-long",
			in:   strings.Repeat("testing.", 256/(len("testing."))) + "example.net",
			ok:   false,
		},
		{
			desc: "hostname",
			in:   "example",
			ok:   true,
		},
		{
			desc: "domain",
			in:   "example.net",
			ok:   true,
		},
		{
			desc: "subdomain",
			in:   "test.example.net",
			ok:   true,
		},
		{
			desc: "dashes",
			in:   "test-with-dash.example.net",
			ok:   true,
		},
		{
			desc: "dash-prefix",
			in:   "-dash-prefix.example.net",
			ok:   false,
		},
		{
			desc: "dash-suffix",
			in:   "dash-suffix-.example.net",
			ok:   false,
		},
		{
			desc: "underscore",
			in:   "under_score.example.net",
			ok:   false,
		},
		{
			desc: "plus",
			in:   "pl+us.example.net",
			ok:   false,
		},
		{
			desc: "brackets",
			in:   "bra[k]ets.example.net",
			ok:   false,
		},
		{
			desc: "parens",
			in:   "pa[re]ns.example.net",
			ok:   false,
		},
		{
			desc: "wild",
			in:   "*.example.net",
			ok:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if ok := isDNS1123Domain(tt.in); ok != tt.ok {
				t.Errorf("isDNS1123Domain(%q); got: %v; want: %v", tt.in, ok, tt.ok)
			}
		})
	}
}
