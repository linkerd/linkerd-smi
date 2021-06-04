package test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/linkerd/linkerd-smi/testutil"
	linkerdtestutil "github.com/linkerd/linkerd2/testutil"
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

const (
	expectedStatColumnCount = 8
)

func TestTrafficSplitWithSMIAdaptor(t *testing.T) {

	// Install Linkerd
	out, err := TestHelper.LinkerdRun("install")
	if err != nil {
		linkerdtestutil.AnnotatedFatal(t, "'linkerd smi install' command failed", err)
	}

	out, err = TestHelper.KubectlApply(out, "")
	if err != nil {
		linkerdtestutil.AnnotatedFatalf(t, "'kubectl apply' command failed",
			"'kubectl apply' command failed\n%s", out)
	}

	// Install Viz extension
	out, err = TestHelper.LinkerdRun("viz", "install")
	if err != nil {
		linkerdtestutil.AnnotatedFatal(t, "'linkerd smi install' command failed", err)
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

	version := "v1alpha1-with-smi-adaptor"
	prefixedNs := version
	ctx := context.Background()

	// Create the Namespace
	err = TestHelper.CreateDataPlaneNamespaceIfNotExists(ctx, prefixedNs, map[string]string{})
	if err != nil {
		linkerdtestutil.AnnotatedFatal(t, "namespace creation failed", err)
	}

	out, err = TestHelper.LinkerdRun("inject", "--manual", "testdata/application.yaml")
	if err != nil {
		linkerdtestutil.AnnotatedFatal(t, "'linkerd inject' command failed", err)
	}

	out, err = TestHelper.KubectlApply(out, prefixedNs)
	if err != nil {
		linkerdtestutil.AnnotatedFatalf(t, "'kubectl apply' command failed",
			"'kubectl apply' command failed\n%s", out)
	}

	TsResourceFile := "testdata/traffic-split-leaf-weights.yaml"
	TsResource, err := linkerdtestutil.ReadFile(TsResourceFile)

	if err != nil {
		linkerdtestutil.AnnotatedFatalf(t, "cannot read updated traffic split resource",
			"cannot read updated traffic split resource: %s, %s", TsResource, err)
	}

	out, err = TestHelper.KubectlApply(TsResource, prefixedNs)
	if err != nil {
		linkerdtestutil.AnnotatedFatalf(t, "failed to update traffic split resource",
			"failed to update traffic split resource: %s\n %s", err, out)
	}

	// wait for deployments to start
	for _, deploy := range []string{"backend", "failing", "slow-cooker"} {
		if err := TestHelper.CheckPods(ctx, prefixedNs, deploy, 1); err != nil {
			if rce, ok := err.(*linkerdtestutil.RestartCountError); ok {
				linkerdtestutil.AnnotatedWarn(t, "CheckPods timed-out", rce)
			} else {
				linkerdtestutil.AnnotatedError(t, "CheckPods timed-out", err)
			}
		}
	}

	t.Run(fmt.Sprintf("ensure traffic is sent to one backend only for %s", version), func(t *testing.T) {
		timeout := 40 * time.Second
		err := TestHelper.RetryFor(timeout, func() error {
			out, err := TestHelper.LinkerdRun("viz", "stat", "deploy", "--namespace", prefixedNs, "--from", "deploy/slow-cooker", "-t", "30s")
			if err != nil {
				return err
			}

			rows, err := parseStatRows(out, 1)
			if err != nil {
				return err
			}

			expectedRows := []*linkerdtestutil.RowStat{
				{
					Name:               "backend",
					Meshed:             "1/1",
					Success:            "100.00%",
					TCPOpenConnections: "1",
				},
			}

			if err := validateRowStats(expectedRows, rows); err != nil {
				return err
			}
			return nil
		})

		if err != nil {
			linkerdtestutil.AnnotatedFatal(t, fmt.Sprintf("timed-out ensuring traffic is sent to one backend only (%s)", timeout), err)
		}
	})

	t.Run(fmt.Sprintf("update traffic split resource with equal weights for %s", version), func(t *testing.T) {

		updatedTsResourceFile := "testdata/updated-traffic-split-leaf-weights.yaml"
		updatedTsResource, err := linkerdtestutil.ReadFile(updatedTsResourceFile)

		if err != nil {
			linkerdtestutil.AnnotatedFatalf(t, "cannot read updated traffic split resource",
				"cannot read updated traffic split resource: %s, %s", updatedTsResource, err)
		}

		out, err := TestHelper.KubectlApply(updatedTsResource, prefixedNs)
		if err != nil {
			linkerdtestutil.AnnotatedFatalf(t, "failed to update traffic split resource",
				"failed to update traffic split resource: %s\n %s", err, out)
		}
	})

	t.Run(fmt.Sprintf("ensure traffic is sent to both backends for %s", version), func(t *testing.T) {
		timeout := 40 * time.Second
		err := TestHelper.RetryFor(timeout, func() error {

			out, err := TestHelper.LinkerdRun("viz", "stat", "deploy", "-n", prefixedNs, "--from", "deploy/slow-cooker", "-t", "30s")
			if err != nil {
				return err
			}

			rows, err := parseStatRows(out, 2)
			if err != nil {
				return err
			}

			expectedRows := []*linkerdtestutil.RowStat{
				{
					Name:               "backend",
					Meshed:             "1/1",
					Success:            "100.00%",
					TCPOpenConnections: "1",
				},
				{
					Name:               "failing",
					Meshed:             "1/1",
					Success:            "0.00%",
					TCPOpenConnections: "1",
				},
			}

			if err := validateRowStats(expectedRows, rows); err != nil {
				return err
			}
			return nil
		})

		if err != nil {
			linkerdtestutil.AnnotatedFatal(t, fmt.Sprintf("timed-out ensuring traffic is sent to both backends (%s)", timeout), err)
		}
	})
}

func validateRowStats(expectedRowStats, actualRowStats []*linkerdtestutil.RowStat) error {

	if len(expectedRowStats) != len(actualRowStats) {
		return fmt.Errorf("Expected number of rows to be %d, but found %d", len(expectedRowStats), len(actualRowStats))
	}

	for i := 0; i < len(expectedRowStats); i++ {
		err := compareRowStat(expectedRowStats[i], actualRowStats[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func compareRowStat(expectedRow, actualRow *linkerdtestutil.RowStat) error {

	if actualRow.Name != expectedRow.Name {
		return fmt.Errorf("Expected name to be '%s', got '%s'",
			expectedRow.Name, actualRow.Name)
	}

	if actualRow.Meshed != expectedRow.Meshed {
		return fmt.Errorf("Expected meshed to be '%s', got '%s'",
			expectedRow.Meshed, actualRow.Meshed)
	}

	if !strings.HasSuffix(actualRow.Rps, "rps") {
		return fmt.Errorf("Unexpected rps for [%s], got [%s]",
			actualRow.Name, actualRow.Rps)
	}

	if !strings.HasSuffix(actualRow.P50Latency, "ms") {
		return fmt.Errorf("Unexpected p50 latency for [%s], got [%s]",
			actualRow.Name, actualRow.P50Latency)
	}

	if !strings.HasSuffix(actualRow.P95Latency, "ms") {
		return fmt.Errorf("Unexpected p95 latency for [%s], got [%s]",
			actualRow.Name, actualRow.P95Latency)
	}

	if !strings.HasSuffix(actualRow.P99Latency, "ms") {
		return fmt.Errorf("Unexpected p99 latency for [%s], got [%s]",
			actualRow.Name, actualRow.P99Latency)
	}

	if actualRow.Success != expectedRow.Success {
		return fmt.Errorf("Expected success to be '%s', got '%s'",
			expectedRow.Success, actualRow.Success)
	}

	if actualRow.TCPOpenConnections != expectedRow.TCPOpenConnections {
		return fmt.Errorf("Expected tcp to be '%s', got '%s'",
			expectedRow.TCPOpenConnections, actualRow.TCPOpenConnections)
	}

	return nil
}

func parseStatRows(out string, expectedRowCount int) ([]*linkerdtestutil.RowStat, error) {
	rows, err := linkerdtestutil.CheckRowCount(out, expectedRowCount)
	if err != nil {
		return nil, err
	}

	var statRows []*linkerdtestutil.RowStat

	for _, row := range rows {
		fields := strings.Fields(row)

		if len(fields) != expectedStatColumnCount {
			return nil, fmt.Errorf(
				"Expected [%d] columns in stat output, got [%d]; full output:\n%s",
				expectedStatColumnCount, len(fields), row)
		}

		row := &linkerdtestutil.RowStat{
			Name:               fields[0],
			Meshed:             fields[1],
			Success:            fields[2],
			Rps:                fields[3],
			P50Latency:         fields[4],
			P95Latency:         fields[5],
			P99Latency:         fields[6],
			TCPOpenConnections: fields[7],
		}

		statRows = append(statRows, row)

	}
	return statRows, nil
}
