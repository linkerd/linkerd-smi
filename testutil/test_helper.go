package testutil

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	serviceprofile "github.com/linkerd/linkerd2/controller/gen/apis/serviceprofile/v1alpha2"
	spclientset "github.com/linkerd/linkerd2/controller/gen/client/clientset/versioned"
	"github.com/linkerd/linkerd2/testutil"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

// TestHelper provides helpers for running the linkerd SMI integration tests.
type TestHelper struct {
	linkerd    string
	namespace  string
	k8sContext string
	spClient   *spclientset.Clientset

	helm
	testutil.KubernetesHelper
}

type helm struct {
	path         string
	smiHelmChart string
	smiVersion   string
}

// NewTestHelper creates a new instance of TestHelper for the current test run.
// The new TestHelper can be configured via command line flags.
func NewTestHelper() *TestHelper {
	exit := func(code int, msg string) {
		fmt.Fprintln(os.Stderr, msg)
		os.Exit(code)
	}

	k8sContext := flag.String("k8s-context", "", "kubernetes context associated with the test cluster")
	linkerd := flag.String("linkerd", "", "path to the linkerd binary to test")
	namespace := flag.String("linkerd-namespace", "linkerd", "the namespace where linkerd is installed")
	verbose := flag.Bool("verbose", false, "turn on debug logging")
	runTests := flag.Bool("integration-tests", false, "must be provided to run the integration tests")
	helmPath := flag.String("helm-path", "", "path to the helm binary to test")
	smiHelmChart := flag.String("smi-helm-chart", "charts/linkerd-smi", "path to linkerd2's SMI extension Helm chart")
	smiHelmVersion := flag.String("smi-helm-version", "", "SMI Version to use when installing linkerd-smi Helm chart")

	flag.Parse()

	if !*runTests {
		exit(0, "integration tests not enabled: enable with -integration-tests")
	}

	if *linkerd == "" {
		exit(1, "-linkerd flag is required")
	}

	if !filepath.IsAbs(*linkerd) {
		exit(1, "-linkerd path must be absolute")
	}

	_, err := os.Stat(*linkerd)
	if err != nil {
		exit(1, "-linkerd binary does not exist")
	}

	if *verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.PanicLevel)
	}

	testHelper := &TestHelper{
		linkerd:    *linkerd,
		namespace:  *namespace,
		k8sContext: *k8sContext,
		helm: helm{
			path:         *helmPath,
			smiHelmChart: *smiHelmChart,
			smiVersion:   *smiHelmVersion,
		},
	}

	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{CurrentContext: *k8sContext}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides)
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		exit(1, fmt.Sprintf("could not read kubernetes config: %s", err.Error()))
	}

	spClient, err := spclientset.NewForConfig(config)
	if err != nil {
		exit(1, fmt.Sprintf("error creating serviceprofile clientset: %s", err.Error()))
	}

	kubernetesHelper, err := testutil.NewKubernetesHelper(*k8sContext, testHelper.RetryFor)
	if err != nil {
		exit(1, fmt.Sprintf("error creating kubernetes helper: %s", err.Error()))
	}
	testHelper.KubernetesHelper = *kubernetesHelper
	testHelper.spClient = spClient

	return testHelper
}

// GetSMIHelmChart returns the path to the linkerd-smi Helm chart
func (h *TestHelper) GetSMIHelmChart() string {
	return h.helm.smiHelmChart
}

// GetSMIHelmVersion returns the version of the linkerd-smi Helm chart
func (h *TestHelper) GetSMIHelmVersion() string {
	return h.helm.smiVersion
}

// IsHelm returns true if Helm path is passed
func (h *TestHelper) IsHelm() bool {
	return h.helm.path != ""
}

// LinkerdSMIRun executes a linkerd SMI command returning its stdout.
func (h *TestHelper) LinkerdSMIRun(arg ...string) (string, error) {
	withParams := append([]string{"smi", "--linkerd-namespace", h.namespace, "--context=" + h.k8sContext}, arg...)
	out, stderr, err := combinedOutput("", h.linkerd, withParams...)
	if err != nil {
		return out, fmt.Errorf("command failed: linkerd smi %s\n%s\n%s", strings.Join(arg, " "), err, stderr)
	}
	return out, nil
}

// GetServiceProfile returns the given ServiceProfile
func (h *TestHelper) GetServiceProfile(ctx context.Context, namespace, name string) (*serviceprofile.ServiceProfile, error) {
	sp, err := h.spClient.LinkerdV1alpha2().ServiceProfiles(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return sp, nil
}

// LinkerdRun executes a linkerd command returning its stdout.
func (h *TestHelper) LinkerdRun(arg ...string) (string, error) {
	out, stderr, err := h.PipeToLinkerdRun("", arg...)
	if err != nil {
		return out, fmt.Errorf("command failed: linkerd %s\n%s\n%s", strings.Join(arg, " "), err, stderr)
	}
	return out, nil
}

// HelmInstall runs the helm install subcommand, with the provided arguments
func (h *TestHelper) HelmInstall(chart, releaseName string, arg ...string) (string, string, error) {
	withParams := append([]string{
		"install",
		releaseName,
		chart,
		"--kube-context", h.k8sContext,
		"--timeout", "60m",
		"--wait",
	}, arg...)
	return combinedOutput("", h.helm.path, withParams...)
}

// HelmRun executes a helm command appended with the --context
func (h *TestHelper) HelmRun(arg ...string) (string, string, error) {
	return h.PipeToHelmRun("", arg...)
}

// PipeToHelmRun executes a Helm command appended with the
// --context flag, and provides a string at Stdin.
func (h *TestHelper) PipeToHelmRun(stdin string, arg ...string) (string, string, error) {
	withParams := append([]string{"--kube-context=" + h.k8sContext}, arg...)
	return combinedOutput(stdin, h.helm.path, withParams...)
}

// PipeToLinkerdRun executes a linkerd command appended with the
// --linkerd-namespace flag, and provides a string at Stdin.
func (h *TestHelper) PipeToLinkerdRun(stdin string, arg ...string) (string, string, error) {
	withParams := append([]string{"--linkerd-namespace", h.namespace, "--context=" + h.k8sContext}, arg...)
	return combinedOutput(stdin, h.linkerd, withParams...)
}

// RetryFor retries a given function every second until the function returns
// without an error, or a timeout is reached. If the timeout is reached, it
// returns the last error received from the function.
func (h *TestHelper) RetryFor(timeout time.Duration, fn func() error) error {
	err := fn()
	if err == nil {
		return nil
	}

	timeoutAfter := time.After(timeout)
	retryAfter := time.Tick(time.Second)

	for {
		select {
		case <-timeoutAfter:
			return err
		case <-retryAfter:
			err = fn()
			if err == nil {
				return nil
			}
		}
	}
}

// combinedOutput executes a shell command and returns the output.
func combinedOutput(stdin string, name string, arg ...string) (string, string, error) {
	command := exec.Command(name, arg...)
	command.Stdin = strings.NewReader(stdin)
	var stderr bytes.Buffer
	command.Stderr = &stderr

	stdout, err := command.Output()
	return string(stdout), stderr.String(), err
}
