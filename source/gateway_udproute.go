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

// NewGatewayUDPRouteSource creates a new Gateway UDPRoute source with the given config.
func NewGatewayUDPRouteSource(ctx context.Context, clients ClientGenerator, config *Config) (Source, error) {
	kubeClient, err := clients.KubeClient()
	if err != nil {
		return nil, err
	}
	groupVersion, err := findServedGatewayRouteGroupVersion(kubeClient.Discovery(), "udproutes")
	if err != nil {
		return nil, err
	}
	if groupVersion == v1.GroupVersion {
		return newGatewayRouteSource(ctx, clients, config, "UDPRoute", func(factory informers.SharedInformerFactory) gatewayRouteInformer {
			return &gatewayUDPRouteV1Informer{factory.Gateway().V1().UDPRoutes()}
		})
	}
	return newGatewayRouteSource(ctx, clients, config, "UDPRoute", func(factory informers.SharedInformerFactory) gatewayRouteInformer {
		return &gatewayUDPRouteV1alpha2Informer{factory.Gateway().V1alpha2().UDPRoutes()}
	})
}

type gatewayUDPRouteV1 struct{ route v1.UDPRoute }

func (route *gatewayUDPRouteV1) Object() kubeObject               { return &route.route }
func (route *gatewayUDPRouteV1) Metadata() *metav1.ObjectMeta     { return &route.route.ObjectMeta }
func (route *gatewayUDPRouteV1) Hostnames() []v1.Hostname         { return nil }
func (route *gatewayUDPRouteV1) ParentRefs() []v1.ParentReference { return route.route.Spec.ParentRefs }
func (route *gatewayUDPRouteV1) Protocol() v1.ProtocolType        { return v1.UDPProtocolType }
func (route *gatewayUDPRouteV1) RouteStatus() v1.RouteStatus      { return route.route.Status.RouteStatus }

type gatewayUDPRouteV1Informer struct {
	gatewayinformersv1.UDPRouteInformer
}

func (informer gatewayUDPRouteV1Informer) List(namespace string, selector labels.Selector) ([]gatewayRoute, error) {
	udpRoutes, err := informer.UDPRouteInformer.Lister().UDPRoutes(namespace).List(selector)
	if err != nil {
		return nil, err
	}
	routes := make([]gatewayRoute, len(udpRoutes))
	for routeIndex, udpRoute := range udpRoutes {
		// List results are supposed to be treated as read-only.
		// We make a shallow copy since we're only interested in setting the TypeMeta.
		clone := *udpRoute
		clone.TypeMeta = metav1.TypeMeta{
			APIVersion: v1.GroupVersion.String(),
			Kind:       "UDPRoute",
		}
		routes[routeIndex] = &gatewayUDPRouteV1{clone}
	}
	return routes, nil
}

type gatewayUDPRouteV1alpha2 struct{ route v1alpha2.UDPRoute }

func (route *gatewayUDPRouteV1alpha2) Object() kubeObject           { return &route.route }
func (route *gatewayUDPRouteV1alpha2) Metadata() *metav1.ObjectMeta { return &route.route.ObjectMeta }
func (route *gatewayUDPRouteV1alpha2) Hostnames() []v1.Hostname     { return nil }
func (route *gatewayUDPRouteV1alpha2) ParentRefs() []v1.ParentReference {
	return route.route.Spec.ParentRefs
}
func (route *gatewayUDPRouteV1alpha2) Protocol() v1.ProtocolType { return v1.UDPProtocolType }
func (route *gatewayUDPRouteV1alpha2) RouteStatus() v1.RouteStatus {
	return route.route.Status.RouteStatus
}

type gatewayUDPRouteV1alpha2Informer struct {
	gatewayinformersv1alpha2.UDPRouteInformer
}

func (informer gatewayUDPRouteV1alpha2Informer) List(namespace string, selector labels.Selector) ([]gatewayRoute, error) {
	udpRoutes, err := informer.UDPRouteInformer.Lister().UDPRoutes(namespace).List(selector)
	if err != nil {
		return nil, err
	}
	routes := make([]gatewayRoute, len(udpRoutes))
	for routeIndex, udpRoute := range udpRoutes {
		// List results are supposed to be treated as read-only.
		// We make a shallow copy since we're only interested in setting the TypeMeta.
		clone := *udpRoute
		clone.TypeMeta = metav1.TypeMeta{
			APIVersion: v1alpha2.GroupVersion.String(),
			Kind:       "UDPRoute",
		}
		routes[routeIndex] = &gatewayUDPRouteV1alpha2{clone}
	}
	return routes, nil
}
