package test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/linkerd/linkerd-smi/testutil"
	serviceprofile "github.com/linkerd/linkerd2/controller/gen/apis/serviceprofile/v1alpha2"
	linkerdtestutil "github.com/linkerd/linkerd2/testutil"
	"k8s.io/apimachinery/pkg/api/resource"
)

//////////////////////
///   TEST SETUP   ///
//////////////////////

var (
	TestHelper *testutil.TestHelper
)

func TestMain(m *testing.M) {
	TestHelper = testutil.NewTestHelper()
	os.Exit(m.Run())
}

//////////////////////
/// TEST EXECUTION ///
//////////////////////

func TestTrafficSplitsConversionWithSMIAdaptor(t *testing.T) {

	ctx := context.Background()
	namespace := "linkerd-smi-app"
	tsName := "backend-traffic-split"
	spName := fmt.Sprintf("backend-svc.%s.svc.cluster.local", namespace)
	backend1 := fmt.Sprintf("backend-svc.%s.svc.cluster.local", namespace)
	backend2 := fmt.Sprintf("failing-svc.%s.svc.cluster.local", namespace)

	// Install Linkerd
	out, err := TestHelper.LinkerdRun("install")
	if err != nil {
		linkerdtestutil.AnnotatedFatal(t, "'linkerd install' command failed", err)
	}

	out, err = TestHelper.KubectlApply(out, "")
	if err != nil {
		linkerdtestutil.AnnotatedFatalf(t, "'kubectl apply' command failed",
			"'kubectl apply' command failed\n%s", out)
	}

	// Install SMI extension
	out, err = TestHelper.LinkerdSMIRun("install")
	if err != nil {
		linkerdtestutil.AnnotatedFatal(t, "'linkerd smi install' command failed", err)
	}

	out, err = TestHelper.KubectlApply(out, "")
	if err != nil {
		linkerdtestutil.AnnotatedFatalf(t, "'kubectl apply' command failed",
			"'kubectl apply' command failed\n%s", out)
	}

	// Create the namespace
	err = TestHelper.CreateDataPlaneNamespaceIfNotExists(ctx, namespace, map[string]string{})
	if err != nil {
		linkerdtestutil.AnnotatedFatalf(t, "Creating namespace failed",
			"Creating namespace failed\n%s", out)
	}

	// Deploy the Application
	out, err = TestHelper.LinkerdRun("inject", "--manual", "testdata/application.yaml")
	if err != nil {
		linkerdtestutil.AnnotatedFatal(t, "'linkerd inject' command failed", err)
	}

	out, err = TestHelper.KubectlApply(out, namespace)
	if err != nil {
		linkerdtestutil.AnnotatedFatalf(t, "'kubectl apply' command failed",
			"'kubectl apply' command failed\n%s", out)
	}

	// Apply TrafficSplit
	TsResourceFile := "testdata/traffic-split-leaf-weights.yaml"
	TsResource, err := linkerdtestutil.ReadFile(TsResourceFile)
	if err != nil {
		linkerdtestutil.AnnotatedFatalf(t, "cannot read updated traffic split resource",
			"cannot read updated traffic split resource: %s, %s", TsResource, err)
	}

	out, err = TestHelper.KubectlApply(TsResource, namespace)
	if err != nil {
		linkerdtestutil.AnnotatedFatalf(t, "failed to update traffic split resource",
			"failed to update traffic split resource: %s\n %s", err, out)
	}

	// Get the resultant ServiceProfile
	var sp *serviceprofile.ServiceProfile
	err = TestHelper.RetryFor(time.Minute, func() error {
		sp, err = TestHelper.GetServiceProfile(ctx, namespace, spName)
		return err
	})
	if err != nil {
		linkerdtestutil.AnnotatedFatalf(t, "failed to retrieve serviceprofile resource",
			"failed to retrieve serviceprofile resource: %s\n %s", err, out)
	}

	// Check if the SP has relevant values
	err = checkIfServiceProfileMatches(sp, spName, namespace, []serviceprofile.WeightedDst{
		{
			Authority: backend1,
			Weight:    resource.MustParse("500m"),
		},
		{
			Authority: backend2,
			Weight:    resource.MustParse("0m"),
		},
	})
	if err != nil {
		linkerdtestutil.AnnotatedFatalf(t, "failed to match serviceprofile resource",
			"failed to match serviceprofile resource: %s", err)
	}

	// Update the TrafficSplit
	TsResourceFile = "testdata/updated-traffic-split-leaf-weights.yaml"
	TsResource, err = linkerdtestutil.ReadFile(TsResourceFile)
	if err != nil {
		linkerdtestutil.AnnotatedFatalf(t, "cannot read updated traffic split resource",
			"cannot read updated traffic split resource: %s, %s", TsResource, err)
	}

	out, err = TestHelper.KubectlApply(TsResource, namespace)
	if err != nil {
		linkerdtestutil.AnnotatedFatalf(t, "failed to update traffic split resource",
			"failed to update traffic split resource: %s\n %s", err, out)
	}

	// Wait for the Controller to sync up the changes
	time.Sleep(10 * time.Second)

	// Check the resultant ServiceProfile
	err = TestHelper.RetryFor(time.Minute, func() error {
		sp, err = TestHelper.GetServiceProfile(ctx, namespace, spName)
		return err
	})
	if err != nil {
		linkerdtestutil.AnnotatedFatalf(t, "failed to retrieve serviceprofile resource",
			"failed to retrieve serviceprofile resource: %s\n %s", err, out)
	}

	// Check if the SP has relevant values
	err = checkIfServiceProfileMatches(sp, spName, namespace, []serviceprofile.WeightedDst{
		{
			Authority: backend1,
			Weight:    resource.MustParse("500m"),
		},
		{
			Authority: backend2,
			Weight:    resource.MustParse("500m"),
		},
	})
	if err != nil {
		linkerdtestutil.AnnotatedFatalf(t, "failed to match serviceprofile resource",
			"failed to match serviceprofile resource: %s", err)
	}

	// Delete the TrafficSplit
	out, err = TestHelper.Kubectl("", "delete", fmt.Sprintf("--namespace=%s", namespace), fmt.Sprintf("trafficsplit/%s", tsName))
	if err != nil {
		linkerdtestutil.AnnotatedFatalf(t, "'kubectl delete' command failed",
			"'kubectl delete' command failed\n%s", out)
	}

	// Wait for the Controller to sync up the changes
	time.Sleep(10 * time.Second)

	// Check the resultant ServiceProfile
	err = TestHelper.RetryFor(time.Minute, func() error {
		sp, err = TestHelper.GetServiceProfile(ctx, namespace, spName)
		return err
	})
	if err != nil {
		linkerdtestutil.AnnotatedFatalf(t, "failed to retrieve serviceprofile resource",
			"failed to retrieve serviceprofile resource: %s\n %s", err, out)
	}

	// Check if the SP has empty values
	err = checkIfServiceProfileMatches(sp, spName, namespace, []serviceprofile.WeightedDst{})
	if err != nil {
		linkerdtestutil.AnnotatedFatalf(t, "failed to match serviceprofile resource",
			"failed to match serviceprofile resource: %s", err)
	}
}

func checkIfServiceProfileMatches(sp *serviceprofile.ServiceProfile, name, namespace string, weightedDsts []serviceprofile.WeightedDst) error {
	if sp.Name != name {
		return fmt.Errorf("Expected serviceprofile.name to be %s but got %s", name, sp.Name)
	}

	if sp.Namespace != namespace {
		return fmt.Errorf("Expected serviceprofile.namespace to be %s but got %s", namespace, sp.Namespace)

	}

	if len(sp.Spec.DstOverrides) != len(weightedDsts) {
		return fmt.Errorf("Expected number of dstoverrides to be %d but got %d", len(weightedDsts), len(sp.Spec.DstOverrides))
	}

	dstOverrides := make(map[string]string)
	for _, dstA := range sp.Spec.DstOverrides {
		dstOverrides[dstA.Authority] = dstA.Weight.String()
	}

	// Check if all the authorties exist
	// in dstOverrides with the same weight
	for _, dst := range weightedDsts {
		weight, ok := dstOverrides[dst.Authority]
		if !ok {
			return fmt.Errorf("Expected service %s to be present in dstOverrides", dst.Authority)
		}

		if weight != dst.Weight.String() {
			return fmt.Errorf("Expected weight to be %s for service %s, but got %s", weight, dst.Authority, dst.Weight.String())
		}
	}

	return nil
}
