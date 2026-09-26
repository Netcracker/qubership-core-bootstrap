package utils

import (
	"context"
	"os"
	"testing"

	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	"github.com/stretchr/testify/assert"
)

// Outside a pod, init must leave the Kubernetes clients unset instead of aborting the process, so
// that packages importing utils can be unit tested.
func TestInit_LeavesClientsUnsetOutsideACluster(t *testing.T) {
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		t.Skip("running inside a Kubernetes cluster")
	}

	assert.Nil(t, K8sClient)
	assert.Nil(t, K8sDynamicClient)
}

func TestLogError_ReturnsTheFormattedError(t *testing.T) {
	err := LogError(logging.GetLogger("test"), context.Background(), "request %s failed: %d", "put", 500)

	assert.EqualError(t, err, "request put failed: 500")
}

// The formatted message is logged as data, so a verb that arrives inside an argument is not
// interpreted a second time.
func TestLogError_KeepsVerbsInsideArguments(t *testing.T) {
	err := LogError(logging.GetLogger("test"), context.Background(), "bad value: %s", "100%d")

	assert.EqualError(t, err, "bad value: 100%d")
}
