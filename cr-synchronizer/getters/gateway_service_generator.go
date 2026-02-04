package getters

import (
	"context"
	"fmt"
	"os"
	"strings"

	k8sv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	gatewayGVRGroup    = "gateway.networking.k8s.io"
	gatewayGVRVersion  = "v1"
	gatewayGVRResource = "gateways"
	httpRouteResource  = "httproutes"
)

// GatewayServiceGenerator applies Service resources conditionally when a Gateway and a referencing HTTPRoute exist.
type GatewayServiceGenerator struct {
	ctx            context.Context
	client         dynamic.Interface
	services       []unstructured.Unstructured
	timeoutSeconds int
}

func NewGatewayServiceGenerator(ctx context.Context, services []unstructured.Unstructured, client dynamic.Interface, timeoutSeconds int) *GatewayServiceGenerator {
	return &GatewayServiceGenerator{
		ctx:            ctx,
		client:         client,
		services:       services,
		timeoutSeconds: timeoutSeconds,
	}
}

func (g *GatewayServiceGenerator) Name() string { return "gatewayServiceApplier" }

func (g *GatewayServiceGenerator) Generate() {
	if val, ok := os.LookupEnv("ISTIO_INTERGATION"); !ok || !strings.EqualFold(val, "true") {
		log.Info().Str("type", "gatewayServiceApplier").Msg("ISTIO_INTERGATION not enabled; skipping gateway service application")
		return
	}

	if len(g.services) == 0 {
		log.Info().Str("type", "gatewayServiceApplier").Msg("no Service objects provided in declaratives, skipping")
		return
	}

	for _, svc := range g.services {
		annotations := svc.GetAnnotations()
		if annotations == nil {
			log.Info().Str("type", "gatewayServiceApplier").Str("service", svc.GetName()).Msg("service has no annotations, skipping")
			continue
		}

		// Annotation keys expected: gateway.target and gateway.route
		gatewayName := strings.TrimSpace(annotations["gateway.target"])
		routeName := strings.TrimSpace(annotations["gateway.route"])
		if gatewayName == "" || routeName == "" {
			log.Info().Str("type", "gatewayServiceApplier").Str("service", svc.GetName()).Msg("required annotations gateway.target or gateway.route missing, skipping")
			continue
		}

		available, err := IsGatewayAndRoutePresent(g.ctx, g.client, gatewayName, routeName)
		if err != nil {
			log.Warn().Str("type", "gatewayServiceApplier").Str("service", svc.GetName()).Err(err).Msg("error checking gateway/route availability, skipping")
			continue
		}
		if !available {
			log.Info().Str("type", "gatewayServiceApplier").Str("service", svc.GetName()).Msg("gateway/route not available, skipping")
			continue
		}

		svcGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}

		customLabels := svc.GetLabels()
		if customLabels == nil {
			customLabels = make(map[string]string)
		}
		customLabels["app.kubernetes.io/managed-by"] = manager
		svc.SetLabels(customLabels)

		_, err = g.client.Resource(svcGVR).Namespace(namespace).Get(g.ctx, svc.GetName(), k8sv1.GetOptions{})
		if err != nil {
			// create
			_, err := g.client.Resource(svcGVR).Namespace(namespace).Create(g.ctx, &svc, k8sv1.CreateOptions{FieldManager: "pre-hook"})
			if err != nil {
				log.Warn().Str("type", "gatewayServiceApplier").Str("service", svc.GetName()).Err(err).Msg("failed to create service")
				continue
			}
			log.Info().Str("type", "gatewayServiceApplier").Str("service", svc.GetName()).Msg("service created")
		} else {
			// update
			svc.SetResourceVersion("")
			_, err := g.client.Resource(svcGVR).Namespace(namespace).Update(g.ctx, &svc, k8sv1.UpdateOptions{FieldManager: "pre-hook"})
			if err != nil {
				log.Warn().Str("type", "gatewayServiceApplier").Str("service", svc.GetName()).Err(err).Msg("failed to update service")
				continue
			}
			log.Info().Str("type", "gatewayServiceApplier").Str("service", svc.GetName()).Msg("service updated")
		}
	}
}

// IsGatewayAndRoutePresent checks whether the named gateway exists and there is an HTTPRoute referencing it (by parentRef name).
func IsGatewayAndRoutePresent(ctx context.Context, client dynamic.Interface, gatewayName, routeName string) (bool, error) {
	gatewayGVR := schema.GroupVersionResource{Group: gatewayGVRGroup, Version: gatewayGVRVersion, Resource: gatewayGVRResource}
	httpRouteGVR := schema.GroupVersionResource{Group: gatewayGVRGroup, Version: gatewayGVRVersion, Resource: httpRouteResource}

	_, err := client.Resource(gatewayGVR).Namespace(namespace).Get(ctx, gatewayName, k8sv1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("gateway %s not found: %w", gatewayName, err)
	}

	list, err := client.Resource(httpRouteGVR).Namespace(namespace).List(ctx, k8sv1.ListOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to list HTTPRoutes: %w", err)
	}

	for _, item := range list.Items {
		if routeName != "" && item.GetName() != routeName {
			continue
		}
		spec, found, err := unstructured.NestedSlice(item.Object, "spec", "parentRefs")
		if err != nil || !found {
			continue
		}
		for _, p := range spec {
			m, ok := p.(map[string]interface{})
			if !ok {
				continue
			}
			if name, ok := m["name"].(string); ok && name == gatewayName {
				if ns, ok := m["namespace"].(string); ok {
					if ns == namespace {
						return true, nil
					}
				} else {
					return true, nil
				}
			}
		}
	}

	return false, nil
}
