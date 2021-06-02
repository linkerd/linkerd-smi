package testutil

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/linkerd/linkerd2/testutil"
	log "github.com/sirupsen/logrus"
)

// TestHelper provides helpers for running the linkerd SMI integration tests.
type TestHelper struct {
	linkerd    string
	namespace  string
	k8sContext string

	testutil.KubernetesHelper
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
	}

	kubernetesHelper, err := testutil.NewKubernetesHelper(*k8sContext, nil)
	if err != nil {
		exit(1, fmt.Sprintf("error creating kubernetes helper: %s", err.Error()))
	}
	testHelper.KubernetesHelper = *kubernetesHelper

	return testHelper
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

// LinkerdRun executes a linkerd command returning its stdout.
func (h *TestHelper) LinkerdRun(arg ...string) (string, error) {
	out, stderr, err := h.PipeToLinkerdRun("", arg...)
	if err != nil {
		return out, fmt.Errorf("command failed: linkerd %s\n%s\n%s", strings.Join(arg, " "), err, stderr)
	}
	return out, nil
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
