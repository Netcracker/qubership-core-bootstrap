package getters

import (
	"context"
	ncv1 "github.com/netcracker/cr-synchronizer/clientset/v1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/watch"
	k8sClientDynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	batchv1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	k8sTesting "k8s.io/client-go/testing"
	"testing"
	"time"
)

type testRecorder struct{}

func (t *testRecorder) LabeledEventf(_ runtime.Object, _ map[string]string, _ map[string]string, _, _, _ string, _ ...interface{}) {
}
func (t *testRecorder) Event(_ runtime.Object, _, _, _ string)                    {}
func (t *testRecorder) Eventf(_ runtime.Object, _, _, _ string, _ ...interface{}) {}
func (t *testRecorder) AnnotatedEventf(_ runtime.Object, _ map[string]string, _, _, _ string, _ ...interface{}) {
}

type fakeClientset struct {
	appsV1 appsv1.AppsV1Interface
}

func (f *fakeClientset) NetcrackerV1() ncv1.V1Alpha1ClientInterface { return nil }
func (f *fakeClientset) CoreV1() corev1.CoreV1Interface             { return nil }
func (f *fakeClientset) AppsV1() appsv1.AppsV1Interface             { return f.appsV1 }
func (f *fakeClientset) BatchV1() batchv1.BatchV1Interface          { return nil }

func TestDeclarationWaiter_UpdatedPhase(t *testing.T) {
	// Reset global watcher map for test isolation
	resourceTypeWatchersMu.Lock()
	resourceTypeWatchers = make(map[schema.GroupVersionResource]*resourceTypeWatcher)
	resourceTypeWatchersMu.Unlock()

	resource := schema.GroupVersionResource{Group: "test", Version: "v1", Resource: "tests"}
	scheme := runtime.NewScheme()
	fclient := k8sClientDynamic.NewSimpleDynamicClient(scheme)
	fakeClientSet := fake.NewSimpleClientset()

	ng := NewDeploymentGenerator(
		context.Background(),
		fclient,
		&testRecorder{},
		&fakeClientset{appsV1: fakeClientSet.AppsV1()},
		scheme,
		&unstructured.Unstructured{},
		false,
		10,
	)

	obj := &unstructured.Unstructured{}
	obj.Object = map[string]interface{}{
		"status": map[string]interface{}{"phase": "Updated"},
	}
	obj.SetName("test-resource")
	fclient.PrependReactor("get", "tests", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, obj, nil
	})

	updatedObj := &unstructured.Unstructured{}
	updatedObj.Object = obj.UnstructuredContent()
	fclient.PrependReactor("update", "tests", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, updatedObj, nil
	})

	deploymentUID := uuid.NewUUID()
	fakeClientSet.PrependReactor("get", "deployments", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &v1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				UID: deploymentUID,
			},
		}, nil
	})

	w := watch.NewFake()
	defer w.Stop()

	fclient.PrependWatchReactor("tests", func(action k8sTesting.Action) (handled bool, ret watch.Interface, err error) {
		return true, w, nil
	})

	done := make(chan struct{})
	go func() {
		ng.declarationWaiter(resource, "test-resource")
		done <- struct{}{}
	}()
	w.Add(obj)

	select {
	case <-done:
		// Success: Now assert owner reference is set
		ownerRefs, found, err := unstructured.NestedSlice(obj.Object, "metadata", "ownerReferences")
		assert.NoError(t, err)
		assert.True(t, found)
		assert.NotEmpty(t, ownerRefs)
		assert.Equal(t, string(deploymentUID), ownerRefs[0].(map[string]interface{})["uid"])
	case <-time.After(1 * time.Second):
		t.Fatal("declarationWaiter did not complete for Updated phase")
	}
}
