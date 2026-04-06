package setup

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

type JWKSManager struct {
	dynamicClient dynamic.Interface
	namespace     string
}

func NewJWKSManager(kubeconfigPath, namespace string) (*JWKSManager, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &JWKSManager{
		dynamicClient: dynamicClient,
		namespace:     namespace,
	}, nil
}

func (m *JWKSManager) UpdateRequestAuthentication(jwks string) error {
	gvr := schema.GroupVersionResource{
		Group:    "security.istio.io",
		Version:  "v1",
		Resource: "requestauthentications",
	}

	unstruct, err := m.dynamicClient.Resource(gvr).Namespace(m.namespace).Get(
		context.Background(), "jwt-auth", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get RequestAuthentication: %w", err)
	}

	spec := unstruct.Object["spec"].(map[string]interface{})
	jwtRules := spec["jwtRules"].([]interface{})
	jwtRules[0].(map[string]interface{})["jwks"] = jwks

	_, err = m.dynamicClient.Resource(gvr).Namespace(m.namespace).Update(
		context.Background(), unstruct, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update RequestAuthentication: %w", err)
	}

	return nil
}
