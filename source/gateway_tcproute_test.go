/*
Copyright 2021 The Kubernetes Authors.

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
	"context"
	"testing"
	"time"

	"sigs.k8s.io/external-dns/internal/testutils"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	kubefake "k8s.io/client-go/kubernetes/fake"
	v1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/gateway-api/apis/v1alpha2"
	gatewayfake "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/fake"

	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/source/annotations"
	templatetest "sigs.k8s.io/external-dns/source/template/testutil"
)

func TestGatewayTCPRouteSourceEndpoints(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	gwClient := gatewayfake.NewSimpleClientset()
	kubeClient := kubefake.NewClientset()
	setFakeGatewayRouteDiscoveryResources(kubeClient, "tcproutes", v1.GroupVersion, v1alpha2.GroupVersion)
	clients := new(testutils.MockClientGenerator)
	clients.On("GatewayClient").Return(gwClient, nil)
	clients.On("KubeClient").Return(kubeClient, nil)

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
	}
	_, err := kubeClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	require.NoError(t, err, "failed to create Namespace")

	ips := []string{"10.64.0.1", "10.64.0.2"}
	gw := &v1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "internal",
			Namespace: "default",
		},
		Spec: v1.GatewaySpec{
			Listeners: []v1.Listener{{
				Protocol: v1.TCPProtocolType,
			}},
		},
		Status: gatewayStatus(ips...),
	}
	_, err = gwClient.GatewayV1().Gateways(gw.Namespace).Create(ctx, gw, metav1.CreateOptions{})
	require.NoError(t, err, "failed to create Gateway")

	rt := &v1.TCPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api",
			Namespace: "default",
			Annotations: map[string]string{
				annotations.HostnameKey: "api-annotation.foobar.internal",
			},
		},
		Spec: v1.TCPRouteSpec{
			CommonRouteSpec: v1.CommonRouteSpec{
				ParentRefs: []v1.ParentReference{
					gwParentRef("default", "internal"),
				},
			},
		},
		Status: v1.TCPRouteStatus{
			RouteStatus: gwRouteStatus(gwParentRef("default", "internal")),
		},
	}
	_, err = gwClient.GatewayV1().TCPRoutes(rt.Namespace).Create(ctx, rt, metav1.CreateOptions{})
	require.NoError(t, err, "failed to create TCPRoute")

	src, err := NewGatewayTCPRouteSource(ctx, clients, &Config{
		TemplateEngine: templatetest.MustEngine(t, "{{.Name}}-template.foobar.internal", "", "", true),
	})
	require.NoError(t, err, "failed to create Gateway TCPRoute Source")

	endpoints, err := src.Endpoints(ctx)
	require.NoError(t, err, "failed to get Endpoints")
	testutils.ValidateEndpoints(t, endpoints, []*endpoint.Endpoint{
		newTestEndpoint("api-annotation.foobar.internal", ips...),
		newTestEndpoint("api-template.foobar.internal", ips...),
	})
}

func TestGatewayTCPRouteSource_InformerTransform(t *testing.T) {
	t.Parallel()

	gwClient := gatewayfake.NewSimpleClientset()
	kubeClient := kubefake.NewClientset()
	setFakeGatewayRouteDiscoveryResources(kubeClient, "tcproutes", v1.GroupVersion)

	rt := &v1.TCPRoute{ObjectMeta: informerTransformObjectMeta()}
	require.Contains(t, rt.GetAnnotations(), corev1.LastAppliedConfigAnnotation)
	require.NotEmpty(t, rt.GetManagedFields())

	_, err := gwClient.GatewayV1().TCPRoutes(rt.GetNamespace()).Create(t.Context(), rt, metav1.CreateOptions{})
	require.NoError(t, err)

	clients := new(testutils.MockClientGenerator)
	clients.On("GatewayClient").Return(gwClient, nil)
	clients.On("KubeClient").Return(kubeClient, nil)

	source, err := NewGatewayTCPRouteSource(t.Context(), clients, &Config{})
	require.NoError(t, err)
	require.IsType(t, &gatewayRouteSource{}, source)
	require.IsType(t, &gatewayTCPRouteV1Informer{}, source.(*gatewayRouteSource).rtInformer)

	testInformerTransformHelper(t,
		source.(*gatewayRouteSource).rtInformer.Informer(),
		rt,
		withRemovedLastAppliedConfigAnnotation(),
		withRemovedManagedFields(),
	)
}

func TestGatewayTCPRouteSourceUsesV1alpha2WhenV1IsNotServed(t *testing.T) {
	t.Parallel()

	legacyRoute := &v1alpha2.TCPRoute{ObjectMeta: metav1.ObjectMeta{Name: "legacy", Namespace: "default"}}
	gatewayClient := gatewayfake.NewSimpleClientset(legacyRoute)
	kubeClient := kubefake.NewClientset()
	setFakeGatewayRouteDiscoveryResources(kubeClient, "tcproutes", v1alpha2.GroupVersion)
	clients := new(testutils.MockClientGenerator)
	clients.On("GatewayClient").Return(gatewayClient, nil)
	clients.On("KubeClient").Return(kubeClient, nil)

	source, err := NewGatewayTCPRouteSource(t.Context(), clients, &Config{})

	require.NoError(t, err)
	require.IsType(t, &gatewayRouteSource{}, source)
	require.IsType(t, &gatewayTCPRouteV1alpha2Informer{}, source.(*gatewayRouteSource).rtInformer)

	routes, err := source.(*gatewayRouteSource).rtInformer.List("default", labels.Everything())
	require.NoError(t, err)
	require.Len(t, routes, 1)
	require.Equal(t, v1alpha2.GroupVersion.String(), routes[0].Object().GetObjectKind().GroupVersionKind().GroupVersion().String())
}

func TestGatewayTCPRouteSourceFiltersNamespaceLabelsAndAnnotations(t *testing.T) {
	t.Parallel()

	testContext, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	parentReference := gwParentRef("default", "internal")
	gateway := &v1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Name: "internal", Namespace: "default"},
		Spec: v1.GatewaySpec{Listeners: []v1.Listener{{
			Protocol: v1.TCPProtocolType,
		}}},
		Status: gatewayStatus("10.64.0.1"),
	}
	buildTCPRoute := func(namespace, name, tier, managedAnnotationValue string) *v1.TCPRoute {
		return &v1.TCPRoute{
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Namespace:   namespace,
				Labels:      map[string]string{"tier": tier},
				Annotations: map[string]string{annotations.HostnameKey: name + ".foobar.internal", "external-dns.kubernetes.io/managed": managedAnnotationValue},
			},
			Spec:   v1.TCPRouteSpec{CommonRouteSpec: v1.CommonRouteSpec{ParentRefs: []v1.ParentReference{parentReference}}},
			Status: v1.TCPRouteStatus{RouteStatus: gwRouteStatus(parentReference)},
		}
	}
	tcpRoutes := []*v1.TCPRoute{
		buildTCPRoute("default", "included", "external", "true"),
		buildTCPRoute("default", "wrong-label", "internal", "true"),
		buildTCPRoute("default", "wrong-annotation", "external", "false"),
		buildTCPRoute("staging", "wrong-namespace", "external", "true"),
	}
	gatewayClient := gatewayfake.NewSimpleClientset()
	_, err := gatewayClient.GatewayV1().Gateways(gateway.Namespace).Create(testContext, gateway, metav1.CreateOptions{})
	require.NoError(t, err)
	for _, tcpRoute := range tcpRoutes {
		_, err = gatewayClient.GatewayV1().TCPRoutes(tcpRoute.Namespace).Create(testContext, tcpRoute, metav1.CreateOptions{})
		require.NoError(t, err)
	}
	kubeClient := kubefake.NewClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "staging"}},
	)
	setFakeGatewayRouteDiscoveryResources(kubeClient, "tcproutes", v1.GroupVersion)
	clients := new(testutils.MockClientGenerator)
	clients.On("GatewayClient").Return(gatewayClient, nil)
	clients.On("KubeClient").Return(kubeClient, nil)

	source, err := NewGatewayTCPRouteSource(testContext, clients, &Config{
		Namespace:        "default",
		LabelFilter:      parseLabelSelectorOrEverything(t, "tier=external"),
		AnnotationFilter: parseLabelSelectorOrEverything(t, "external-dns.kubernetes.io/managed=true"),
	})
	require.NoError(t, err)

	gatewaySource := source.(*gatewayRouteSource)
	filteredRoutes, err := gatewaySource.rtInformer.List(gatewaySource.rtNamespace, gatewaySource.rtLabels)
	require.NoError(t, err)
	require.Len(t, filteredRoutes, 2)
	matchingAnnotationCount := 0
	for _, filteredRoute := range filteredRoutes {
		if gatewaySource.rtAnnotations.Matches(labels.Set(filteredRoute.Metadata().Annotations)) {
			matchingAnnotationCount++
		}
	}
	require.Equal(t, 1, matchingAnnotationCount)

	endpoints, err := source.Endpoints(testContext)

	require.NoError(t, err)
	testutils.ValidateEndpoints(t, endpoints, []*endpoint.Endpoint{newTestEndpoint("included.foobar.internal", "10.64.0.1")})
}
