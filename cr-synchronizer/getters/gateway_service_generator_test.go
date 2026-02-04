package getters

import (
	"context"
	"os"
	"testing"

	k8sv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestServiceAppliedWhenGatewayAndRouteExist(t *testing.T) {
	ns := "test-ns"
	namespace = ns
	ctx := context.Background()
	os.Setenv("ISTIO_INTERGATION", "true")
	defer os.Unsetenv("ISTIO_INTERGATION")

	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme.Scheme, map[schema.GroupVersionResource]string{
		{Group: gatewayGVRGroup, Version: gatewayGVRVersion, Resource: gatewayGVRResource}: "GatewayList",
		{Group: gatewayGVRGroup, Version: gatewayGVRVersion, Resource: httpRouteResource}:  "HTTPRouteList",
		{Group: "", Version: "v1", Resource: "services"}:                                   "ServiceList",
	})

	gw := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "gateway.networking.k8s.io/v1",
		"kind":       "Gateway",
		"metadata": map[string]interface{}{
			"name": "istio-gateway",
		},
	}}

	if _, err := client.Resource(schema.GroupVersionResource{Group: gatewayGVRGroup, Version: gatewayGVRVersion, Resource: gatewayGVRResource}).Namespace(ns).Create(ctx, gw, k8sv1.CreateOptions{}); err != nil {
		t.Fatalf("failed to create gateway: %v", err)
	}

	route := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "gateway.networking.k8s.io/v1",
		"kind":       "HTTPRoute",
		"metadata": map[string]interface{}{
			"name": "fb-route",
		},
		"spec": map[string]interface{}{
			"parentRefs": []interface{}{map[string]interface{}{"name": "istio-gateway"}},
		},
	}}

	if _, err := client.Resource(schema.GroupVersionResource{Group: gatewayGVRGroup, Version: gatewayGVRVersion, Resource: httpRouteResource}).Namespace(ns).Create(ctx, route, k8sv1.CreateOptions{}); err != nil {
		t.Fatalf("failed to create httproute: %v", err)
	}

	svc := unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata": map[string]interface{}{
			"name": "test-svc",
			"annotations": map[string]interface{}{
				"gateway.target": "istio-gateway",
				"gateway.route":  "fb-route",
			},
		},
		"spec": map[string]interface{}{
			"ports": []interface{}{map[string]interface{}{"protocol": "TCP", "port": int64(8080), "targetPort": int64(8080)}},
		},
	}}

	g := NewGatewayServiceGenerator(ctx, []unstructured.Unstructured{svc}, client, 10)
	g.Generate()

	svcGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
	if _, err := client.Resource(svcGVR).Namespace(ns).Get(ctx, "test-svc", k8sv1.GetOptions{}); err != nil {
		t.Fatalf("expected service to be created, got error: %v", err)
	}
}

func TestServiceNotAppliedWhenEnvDisabled(t *testing.T) {
	ns := "test-ns"
	namespace = ns
	ctx := context.Background()
	os.Setenv("ISTIO_INTERGATION", "false")
	defer os.Unsetenv("ISTIO_INTERGATION")

	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme.Scheme, map[schema.GroupVersionResource]string{
		{Group: gatewayGVRGroup, Version: gatewayGVRVersion, Resource: gatewayGVRResource}: "GatewayList",
		{Group: gatewayGVRGroup, Version: gatewayGVRVersion, Resource: httpRouteResource}:  "HTTPRouteList",
		{Group: "", Version: "v1", Resource: "services"}:                                   "ServiceList",
	})

	gw := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "gateway.networking.k8s.io/v1",
		"kind":       "Gateway",
		"metadata": map[string]interface{}{
			"name": "istio-gateway",
		},
	}}

	if _, err := client.Resource(schema.GroupVersionResource{Group: gatewayGVRGroup, Version: gatewayGVRVersion, Resource: gatewayGVRResource}).Namespace(ns).Create(ctx, gw, k8sv1.CreateOptions{}); err != nil {
		t.Fatalf("failed to create gateway: %v", err)
	}

	route := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "gateway.networking.k8s.io/v1",
		"kind":       "HTTPRoute",
		"metadata": map[string]interface{}{
			"name": "fb-route",
		},
		"spec": map[string]interface{}{
			"parentRefs": []interface{}{map[string]interface{}{"name": "istio-gateway"}},
		},
	}}

	if _, err := client.Resource(schema.GroupVersionResource{Group: gatewayGVRGroup, Version: gatewayGVRVersion, Resource: httpRouteResource}).Namespace(ns).Create(ctx, route, k8sv1.CreateOptions{}); err != nil {
		t.Fatalf("failed to create httproute: %v", err)
	}

	svc := unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata": map[string]interface{}{
			"name": "test-svc",
			"annotations": map[string]interface{}{
				"gateway.target": "istio-gateway",
				"gateway.route":  "fb-route",
			},
		},
		"spec": map[string]interface{}{
			"ports": []interface{}{map[string]interface{}{"protocol": "TCP", "port": int64(8080), "targetPort": int64(8080)}},
		},
	}}

	g := NewGatewayServiceGenerator(ctx, []unstructured.Unstructured{svc}, client, 10)
	g.Generate()

	svcGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
	if _, err := client.Resource(svcGVR).Namespace(ns).Get(ctx, "test-svc", k8sv1.GetOptions{}); err == nil {
		t.Fatalf("expected service NOT to be created since ISTIO_INTERGATION is false")
	}
}

func TestServiceNotAppliedWhenRouteMissing(t *testing.T) {
	ns := "test-ns"
	namespace = ns
	ctx := context.Background()
	os.Setenv("ISTIO_INTERGATION", "true")
	defer os.Unsetenv("ISTIO_INTERGATION")

	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme.Scheme, map[schema.GroupVersionResource]string{
		{Group: gatewayGVRGroup, Version: gatewayGVRVersion, Resource: gatewayGVRResource}: "GatewayList",
		{Group: gatewayGVRGroup, Version: gatewayGVRVersion, Resource: httpRouteResource}:  "HTTPRouteList",
		{Group: "", Version: "v1", Resource: "services"}:                                   "ServiceList",
	})

	gw := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "gateway.networking.k8s.io/v1",
		"kind":       "Gateway",
		"metadata": map[string]interface{}{
			"name": "istio-gateway",
		},
	}}

	if _, err := client.Resource(schema.GroupVersionResource{Group: gatewayGVRGroup, Version: gatewayGVRVersion, Resource: gatewayGVRResource}).Namespace(ns).Create(ctx, gw, k8sv1.CreateOptions{}); err != nil {
		t.Fatalf("failed to create gateway: %v", err)
	}

	svc := unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata": map[string]interface{}{
			"name": "test-svc",
			"annotations": map[string]interface{}{
				"gateway.target": "istio-gateway",
				"gateway.route":  "fb-route",
			},
		},
		"spec": map[string]interface{}{
			"ports": []interface{}{map[string]interface{}{"protocol": "TCP", "port": int64(8080), "targetPort": int64(8080)}},
		},
	}}

	g := NewGatewayServiceGenerator(ctx, []unstructured.Unstructured{svc}, client, 10)
	g.Generate()

	svcGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
	if _, err := client.Resource(svcGVR).Namespace(ns).Get(ctx, "test-svc", k8sv1.GetOptions{}); err == nil {
		t.Fatalf("expected service NOT to be created since route is missing")
	}
}
