package foobar

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	appsv1alpha1 "github.com/undeadops/fooOperator/api/v1alpha1"
	"github.com/undeadops/fooOperator/controllers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/undeadops/fooOperator/pkg/hash"
)

type FooBarAdapter struct {
	fooBar      *appsv1alpha1.FooBar
	currentFoos []*appsv1alpha1.Foo
	oldFoos     []*appsv1alpha1.Foo
	logger      logr.Logger
	client      client.Client
	specHash    string
	scheme      *runtime.Scheme
	recorder    record.EventRecorder
}

type ObjectState bool

const (
	ObjectModified  ObjectState = true
	ObjectUnchanged ObjectState = false
)

const FooFinalizer string = "foobar.undeadops.xyz/finalizer"
const FooHashLabel string = "foo-hash"

func NewFooBarAdapter(foobar *appsv1alpha1.FooBar, logger logr.Logger, client client.Client,
	scheme *runtime.Scheme, recorder record.EventRecorder) *FooBarAdapter {
	adapter := &FooBarAdapter{
		fooBar:   foobar,
		logger:   logger,
		client:   client,
		scheme:   scheme,
		recorder: recorder,
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(3))
	defer cancel()

	// Generate current spec hash
	adapter.specHash = hash.ComputeTemplateHash(&foobar.Spec.Foo, nil)

	// Get List of all current/old foos
	err := adapter.GetFoos(ctx)
	if err != nil {
		adapter.logger.Error(err, "Unable to retrieve old Foos")
	}

	return adapter
}

func (a *FooBarAdapter) GetFoos(ctx context.Context) error {
	foos := &appsv1alpha1.FooList{}
	err := a.client.List(ctx, foos)
	if err != nil {
		return err
	}

	var foosCurrent []*appsv1alpha1.Foo
	var foosOld []*appsv1alpha1.Foo

	for _, foo := range foos.Items {
		labels := foo.GetLabels()
		if metav1.IsControlledBy(&foo, a.fooBar) {
			if labels[FooHashLabel] == a.specHash {
				foosCurrent = append(foosCurrent, &foo)
			} else {
				foosOld = append(foosOld, &foo)
			}
		}
	}

	a.currentFoos = foosCurrent
	a.oldFoos = foosOld
	return nil
}

func (a *FooBarAdapter) UpdateStatus() (controllers.OperationResult, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(2))
	defer cancel()

	idleFoos := int32(0)
	activeFoos := int32(0)

	for _, f := range a.currentFoos {
		if f.Status.State == "Idle" {
			idleFoos++
		}
		activeFoos++
	}

	status := appsv1alpha1.FooBarStatus{
		Idle:        idleFoos,
		Active:      activeFoos,
		CurrentHash: a.specHash,
	}

	if !reflect.DeepEqual(a.fooBar.Status, status) {
		a.fooBar.Status = status
		err := a.client.Status().Update(ctx, a.fooBar)
		if err != nil {
			return controllers.Requeue()
		}
		return controllers.StopProcessing()
	}
	return controllers.ContinueProcessing()
}

func (a *FooBarAdapter) ScaleUp() (controllers.OperationResult, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(2))
	defer cancel()

	if a.fooBar.Status.Active < a.fooBar.Spec.Replicas {
		a.logger.Info(fmt.Sprintf("Creating Foo for Replicas: %d < %d", len(a.currentFoos), a.fooBar.Spec.Replicas))
		foo := a.newFoo()
		err := a.client.Create(ctx, foo)
		if err != nil {
			a.recorder.Eventf(a.fooBar, corev1.EventTypeWarning, "FooCreate",
				"Failed to Create Foo %s/%s", a.fooBar.GetNamespace(), foo.GetName())
			return controllers.RequeueWithError(fmt.Errorf("failed to create Foo from FooBar ScaleUp: %v", err))
		}
		a.recorder.Eventf(a.fooBar, corev1.EventTypeNormal, "FooCreate",
			"Successfully Created Foo %s/%s", a.fooBar.GetNamespace(), foo.GetName())
		return controllers.StopProcessing()
	}
	return controllers.ContinueProcessing()
}

func (a *FooBarAdapter) ScaleDown() (controllers.OperationResult, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(2))
	defer cancel()

	var err error

	if a.fooBar.Status.Active > a.fooBar.Spec.Replicas {
		a.logger.Info(fmt.Sprintf("Deleting Foo for replica count: %d > %d", len(a.currentFoos), a.fooBar.Spec.Replicas))

		// Get Oldest Foo to delete
		oldFoo := a.GetOldestFoo(ctx)

		foo := &appsv1alpha1.Foo{}
		err = a.client.Get(ctx, types.NamespacedName{Namespace: oldFoo.GetNamespace(), Name: oldFoo.GetName()}, foo)
		if err != nil {
			if errors.IsNotFound(err) {
				return controllers.RequeueWithError(fmt.Errorf("failed to retrieve Foo for ScaledDown Delete: %v", err))
			}
			return controllers.RequeueOnErrorOrStop(err)
		}

		err = a.client.Delete(ctx, foo)
		if err != nil {
			a.recorder.Eventf(a.fooBar, corev1.EventTypeWarning, "ScaleDown",
				"Failed to Delete old Foo %s/%s", a.fooBar.GetNamespace(), foo.GetName())
			return controllers.RequeueWithError(fmt.Errorf("failed to delete old foo from FooBar Scaledown: %v", err))
		}
		a.recorder.Eventf(a.fooBar, corev1.EventTypeNormal, "ScaleDown",
			"Successfully Removed Foo %s/%s", a.fooBar.GetNamespace(), foo.GetName())
		return controllers.StopProcessing()
	}
	return controllers.ContinueProcessing()
}

func (a *FooBarAdapter) newFoo() *appsv1alpha1.Foo {
	labels := a.fooBar.DeepCopy().Labels
	labels[FooHashLabel] = a.specHash

	foo := &appsv1alpha1.Foo{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-%s-%s", a.fooBar.GetName(), a.specHash, rand.String(6)),
			Namespace:   a.fooBar.GetNamespace(),
			Labels:      labels,
			Annotations: a.fooBar.Annotations,
		},
		Spec: a.fooBar.Spec.DeepCopy().Foo,
	}

	// Ensure FooBar is parent to our Foos
	_ = ctrl.SetControllerReference(a.fooBar, foo, a.scheme)
	return foo
}

func (a *FooBarAdapter) GetOldestFoo(ctx context.Context) *appsv1alpha1.Foo {
	old := a.currentFoos[0]
	oldDate := a.currentFoos[0].GetCreationTimestamp()

	for _, f := range a.oldFoos {
		if f.CreationTimestamp.Before(&oldDate) {
			old = f
			oldDate = f.GetCreationTimestamp()
		}
	}

	for _, f := range a.currentFoos {
		if f.CreationTimestamp.Before(&oldDate) {
			old = f
			oldDate = f.GetCreationTimestamp()
		}
	}

	return old
}
