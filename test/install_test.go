package test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/linkerd/linkerd-smi/testutil"
	serviceprofile "github.com/linkerd/linkerd2/controller/gen/apis/serviceprofile/v1alpha2"
	"github.com/linkerd/linkerd2/pkg/tls"
	linkerdtestutil "github.com/linkerd/linkerd2/testutil"
	"k8s.io/apimachinery/pkg/api/resource"
)

//////////////////////
///   TEST SETUP   ///
//////////////////////

var (
	TestHelper *testutil.TestHelper
	namespace  = "linkerd-smi-app"
	tsName     = "backend-traffic-split"
	spName     = fmt.Sprintf("backend-svc.%s.svc.cluster.local", namespace)
	backend1   = fmt.Sprintf("backend-svc.%s.svc.cluster.local", namespace)
	backend2   = fmt.Sprintf("failing-svc.%s.svc.cluster.local", namespace)
)

func TestMain(m *testing.M) {
	TestHelper = testutil.NewTestHelper()
	os.Exit(m.Run())
}

//////////////////////
/// TEST EXECUTION ///
//////////////////////

func TestSMIAdaptorWithCLI(t *testing.T) {
	// TODO: Skip if Helm path is passed, Better toggling?
	if TestHelper.IsHelm() {
		return
	}

	ctx := context.Background()

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

	out, err = TestHelper.Kubectl("", "--namespace=linkerd", "rollout", "status", "--timeout=60m", "deploy/linkerd-destination")
	if err != nil {
		linkerdtestutil.AnnotatedFatalf(t,
			"failed to wait rollout of deploy/linkerd-destination",
			"failed to wait for rollout of deploy/linkerd-destination: %s: %s", err, out)
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

	out, err = TestHelper.LinkerdSMIRun("check")
	if err != nil {
		linkerdtestutil.AnnotatedFatalf(t, "'linkerd smi check' command failed",
			"'linkerd smi check' command failed\n%s", out)
	}

	if err = testTrafficSplitsConversionWithSMIAdaptor(ctx); err != nil {
		linkerdtestutil.AnnotatedFatalf(t,
			"failed to test SMI Adaptor",
			"failed to test SMI Adaptor: %s", err)
	}
}

func TestSMIAdaptorWithHelm(t *testing.T) {

	// Skip if no Helm path is passed
	if !TestHelper.IsHelm() {
		return
	}

	ctx := context.Background()

	// Install Linkerd Edge
	_, _, err := TestHelper.HelmRun("repo", "add", "linkerd-edge", "https://helm.linkerd.io/edge")
	if err != nil {
		linkerdtestutil.AnnotatedFatal(t, "'helm repo add' command failed", err)
	}

	helmTLSCerts, err := tls.GenerateRootCAWithDefaults("identity.linkerd.cluster.local")
	if err != nil {
		linkerdtestutil.AnnotatedFatalf(t, "failed to generate root certificate for identity",
			"failed to generate root certificate for identity: %s", err)
	}

	args := []string{
		"--set", "identityTrustAnchorsPEM=" + helmTLSCerts.Cred.Crt.EncodeCertificatePEM(),
		"--set", "identity.issuer.tls.crtPEM=" + helmTLSCerts.Cred.Crt.EncodeCertificatePEM(),
		"--set", "identity.issuer.tls.keyPEM=" + helmTLSCerts.Cred.EncodePrivateKeyPEM(),
		"--set", "identity.issuer.crtExpiry=" + helmTLSCerts.Cred.Crt.Certificate.NotAfter.Format(time.RFC3339),
	}

	if stdout, stderr, err := TestHelper.HelmInstall("linkerd-edge/linkerd2", "linkerd", args...); err != nil {
		linkerdtestutil.AnnotatedFatalf(t, "'helm install' command failed\n%s\n%s\n%v", stdout, stderr, err)
	}

	// Check if linkerd is ready
	for _, deploy := range []string{"linkerd-destination", "linkerd-identity", "linkerd-proxy-injector"} {
		if err := TestHelper.CheckPods(ctx, "linkerd", deploy, 1); err != nil {
			if rce, ok := err.(*linkerdtestutil.RestartCountError); ok {
				linkerdtestutil.AnnotatedFatal(t, "CheckPods timed-out", rce)
			} else {
				linkerdtestutil.AnnotatedFatal(t, "CheckPods timed-out", err)
			}
		}
	}

	// Install SMI Extension
	// Use the version if it is passed
	var smiArgs []string
	if TestHelper.GetSMIHelmVersion() != "" {
		smiArgs = append(smiArgs, []string{
			"--set", "adaptor.image.tag=" + TestHelper.GetSMIHelmVersion(),
			"--namespace", TestHelper.GetSMINamespace(),
			"--create-namespace",
		}...)
	}

	// Set namespace creation flags
	smiArgs = append(smiArgs, []string{
		"--namespace", TestHelper.GetSMINamespace(),
		"--create-namespace",
	}...)

	if stdout, stderr, err := TestHelper.HelmInstall(TestHelper.GetSMIHelmChart(), "linkerd-smi", smiArgs...); err != nil {
		linkerdtestutil.AnnotatedFatalf(t, "'helm install' command failed\n%s\n%s\n%v", stdout, stderr, err)
	}

	o, err := TestHelper.Kubectl("", "--namespace=linkerd-smi", "rollout", "status", "--timeout=60m", "deploy/smi-adaptor")
	if err != nil {
		linkerdtestutil.AnnotatedFatalf(t,
			"failed to wait rollout of deploy/smi-adaptor",
			"failed to wait for rollout of deploy/smi-adaptor: %s: %s", err, o)
	}

	if err = testTrafficSplitsConversionWithSMIAdaptor(ctx); err != nil {
		linkerdtestutil.AnnotatedFatalf(t,
			"failed to test SMI Adaptor",
			"failed to test SMI Adaptor: %s", err)
	}
}

func testTrafficSplitsConversionWithSMIAdaptor(ctx context.Context) error {

	// Create the namespace
	err := TestHelper.CreateDataPlaneNamespaceIfNotExists(ctx, namespace, map[string]string{})
	if err != nil {
		return fmt.Errorf("Creating namespace failed: %s", err)
	}

	// Deploy the Application
	out, err := TestHelper.LinkerdRun("inject", "--manual", "testdata/application.yaml")
	if err != nil {
		return fmt.Errorf("'linkerd inject' command failed: %s", err)
	}

	out, err = TestHelper.KubectlApply(out, namespace)
	if err != nil {
		return fmt.Errorf("'kubectl apply' command failed\n%s", out)
	}

	// Apply TrafficSplit
	TsResourceFile := "testdata/traffic-split-leaf-weights.yaml"
	TsResource, err := linkerdtestutil.ReadFile(TsResourceFile)
	if err != nil {
		return fmt.Errorf("cannot read updated traffic split resource: %s, %s", TsResource, err)
	}

	out, err = TestHelper.KubectlApply(TsResource, namespace)
	if err != nil {
		return fmt.Errorf("failed to update traffic split resource: %s\n %s", err, out)
	}

	// Get the resultant ServiceProfile
	var sp *serviceprofile.ServiceProfile
	err = TestHelper.RetryFor(2*time.Minute, func() error {
		sp, err = TestHelper.GetServiceProfile(ctx, namespace, spName)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to retrieve serviceprofile resource: %s\n %s", err, out)
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
		return fmt.Errorf("failed to match serviceprofile resource: %s", err)
	}

	// Update the TrafficSplit
	TsResourceFile = "testdata/updated-traffic-split-leaf-weights.yaml"
	TsResource, err = linkerdtestutil.ReadFile(TsResourceFile)
	if err != nil {
		return fmt.Errorf("cannot read updated traffic split resource: %s, %s", TsResource, err)
	}

	out, err = TestHelper.KubectlApply(TsResource, namespace)
	if err != nil {
		return fmt.Errorf("failed to update traffic split resource: %s\n %s", err, out)
	}

	// Wait for the Controller to sync up the changes
	time.Sleep(10 * time.Second)

	// Check the resultant ServiceProfile
	err = TestHelper.RetryFor(time.Minute, func() error {
		sp, err = TestHelper.GetServiceProfile(ctx, namespace, spName)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to retrieve serviceprofile resource: %s\n %s", err, out)
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
		return fmt.Errorf("failed to match serviceprofile resource: %s", err)
	}

	// Delete the TrafficSplit
	out, err = TestHelper.Kubectl("", "delete", fmt.Sprintf("--namespace=%s", namespace), fmt.Sprintf("trafficsplit/%s", tsName))
	if err != nil {
		return fmt.Errorf("'kubectl delete' command failed\n%s", out)
	}

	// Wait for the Controller to sync up the changes
	time.Sleep(10 * time.Second)

	// Check the resultant ServiceProfile
	err = TestHelper.RetryFor(time.Minute, func() error {
		sp, err = TestHelper.GetServiceProfile(ctx, namespace, spName)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to retrieve serviceprofile resource: %s\n %s", err, out)
	}

	// Check if the SP has empty values
	err = checkIfServiceProfileMatches(sp, spName, namespace, []serviceprofile.WeightedDst{})
	if err != nil {
		return fmt.Errorf("failed to match serviceprofile resource: %s", err)
	}

	return nil
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
