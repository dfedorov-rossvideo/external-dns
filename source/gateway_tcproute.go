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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	v1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/gateway-api/apis/v1alpha2"
	informers "sigs.k8s.io/gateway-api/pkg/client/informers/externalversions"
	gatewayinformersv1 "sigs.k8s.io/gateway-api/pkg/client/informers/externalversions/apis/v1"
	gatewayinformersv1alpha2 "sigs.k8s.io/gateway-api/pkg/client/informers/externalversions/apis/v1alpha2"
)

// NewGatewayTCPRouteSource creates a new Gateway TCPRoute source with the given config.
func NewGatewayTCPRouteSource(ctx context.Context, clients ClientGenerator, config *Config) (Source, error) {
	kubeClient, err := clients.KubeClient()
	if err != nil {
		return nil, err
	}
	groupVersion, err := findServedGatewayRouteGroupVersion(kubeClient.Discovery(), "tcproutes")
	if err != nil {
		return nil, err
	}
	if groupVersion == v1.GroupVersion {
		return newGatewayRouteSource(ctx, clients, config, "TCPRoute", func(factory informers.SharedInformerFactory) gatewayRouteInformer {
			return &gatewayTCPRouteV1Informer{factory.Gateway().V1().TCPRoutes()}
		})
	}
	return newGatewayRouteSource(ctx, clients, config, "TCPRoute", func(factory informers.SharedInformerFactory) gatewayRouteInformer {
		return &gatewayTCPRouteV1alpha2Informer{factory.Gateway().V1alpha2().TCPRoutes()}
	})
}

type gatewayTCPRouteV1 struct{ route v1.TCPRoute }

func (route *gatewayTCPRouteV1) Object() kubeObject               { return &route.route }
func (route *gatewayTCPRouteV1) Metadata() *metav1.ObjectMeta     { return &route.route.ObjectMeta }
func (route *gatewayTCPRouteV1) Hostnames() []v1.Hostname         { return nil }
func (route *gatewayTCPRouteV1) ParentRefs() []v1.ParentReference { return route.route.Spec.ParentRefs }
func (route *gatewayTCPRouteV1) Protocol() v1.ProtocolType        { return v1.TCPProtocolType }
func (route *gatewayTCPRouteV1) RouteStatus() v1.RouteStatus      { return route.route.Status.RouteStatus }

type gatewayTCPRouteV1Informer struct {
	gatewayinformersv1.TCPRouteInformer
}

func (informer gatewayTCPRouteV1Informer) List(namespace string, selector labels.Selector) ([]gatewayRoute, error) {
	tcpRoutes, err := informer.TCPRouteInformer.Lister().TCPRoutes(namespace).List(selector)
	if err != nil {
		return nil, err
	}
	routes := make([]gatewayRoute, len(tcpRoutes))
	for routeIndex, tcpRoute := range tcpRoutes {
		// List results are supposed to be treated as read-only.
		// We make a shallow copy since we're only interested in setting the TypeMeta.
		clone := *tcpRoute
		clone.TypeMeta = metav1.TypeMeta{
			APIVersion: v1.GroupVersion.String(),
			Kind:       "TCPRoute",
		}
		routes[routeIndex] = &gatewayTCPRouteV1{clone}
	}
	return routes, nil
}

type gatewayTCPRouteV1alpha2 struct{ route v1alpha2.TCPRoute }

func (route *gatewayTCPRouteV1alpha2) Object() kubeObject           { return &route.route }
func (route *gatewayTCPRouteV1alpha2) Metadata() *metav1.ObjectMeta { return &route.route.ObjectMeta }
func (route *gatewayTCPRouteV1alpha2) Hostnames() []v1.Hostname     { return nil }
func (route *gatewayTCPRouteV1alpha2) ParentRefs() []v1.ParentReference {
	return route.route.Spec.ParentRefs
}
func (route *gatewayTCPRouteV1alpha2) Protocol() v1.ProtocolType { return v1.TCPProtocolType }
func (route *gatewayTCPRouteV1alpha2) RouteStatus() v1.RouteStatus {
	return route.route.Status.RouteStatus
}

type gatewayTCPRouteV1alpha2Informer struct {
	gatewayinformersv1alpha2.TCPRouteInformer
}

func (informer gatewayTCPRouteV1alpha2Informer) List(namespace string, selector labels.Selector) ([]gatewayRoute, error) {
	tcpRoutes, err := informer.TCPRouteInformer.Lister().TCPRoutes(namespace).List(selector)
	if err != nil {
		return nil, err
	}
	routes := make([]gatewayRoute, len(tcpRoutes))
	for routeIndex, tcpRoute := range tcpRoutes {
		// List results are supposed to be treated as read-only.
		// We make a shallow copy since we're only interested in setting the TypeMeta.
		clone := *tcpRoute
		clone.TypeMeta = metav1.TypeMeta{
			APIVersion: v1alpha2.GroupVersion.String(),
			Kind:       "TCPRoute",
		}
		routes[routeIndex] = &gatewayTCPRouteV1alpha2{clone}
	}
	return routes, nil
}
