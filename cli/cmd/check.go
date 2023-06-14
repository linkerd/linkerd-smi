package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/linkerd/linkerd2/pkg/healthcheck"
	"github.com/spf13/cobra"
)

const (

	// linkerdSMIExtensionCheck adds checks related to the SMI extension
	linkerdSMIExtensionCheck healthcheck.CategoryID = "linkerd-smi"
)

type checkOptions struct {
	wait      time.Duration
	output    string
	proxy     bool
	namespace string
	pre       string
}

func smiCategory(hc *healthcheck.HealthChecker) *healthcheck.Category {

	checkers := []healthcheck.Checker{}

	checkers = append(checkers,
		*healthcheck.NewChecker("linkerd-smi extension Namespace exists").
			WithHintAnchor("l5d-smi-ns-exists").
			Fatal().
			WithCheck(func(ctx context.Context) error {
				// Get SMI Extension Namespace
				_, err := hc.KubeAPIClient().GetNamespaceWithExtensionLabel(ctx, smiExtensionName)
				if err != nil {
					return err
				}
				return nil
			}))

	checkers = append(checkers,
		*healthcheck.NewChecker("SMI extension service account exists").
			WithHintAnchor("l5d-smi-sc-exists").
			Fatal().
			Warning().
			WithCheck(func(ctx context.Context) error {
				// Check for Collector Service Account
				return healthcheck.CheckServiceAccounts(ctx, hc.KubeAPIClient(), []string{"smi-adaptor"}, hc.DataPlaneNamespace, "")
			}))

	checkers = append(checkers,
		*healthcheck.NewChecker("SMI extension pods are injected").
			WithHintAnchor("l5d-smi-pods-injection").
			Warning().
			WithCheck(func(ctx context.Context) error {
				// Check if SMI Extension pods have been injected
				pods, err := hc.KubeAPIClient().GetPodsByNamespace(ctx, hc.DataPlaneNamespace)
				if err != nil {
					return err
				}
				return healthcheck.CheckIfDataPlanePodsExist(pods)
			}))

	checkers = append(checkers,
		*healthcheck.NewChecker("SMI extension pods are running").
			WithHintAnchor("l5d-smi-pods-running").
			Fatal().
			WithRetryDeadline(hc.RetryDeadline).
			SurfaceErrorOnRetry().
			WithCheck(func(ctx context.Context) error {
				pods, err := hc.KubeAPIClient().GetPodsByNamespace(ctx, hc.DataPlaneNamespace)
				if err != nil {
					return err
				}

				// Check for relevant pods to be present
				err = healthcheck.CheckForPods(pods, []string{"smi-adaptor"})
				if err != nil {
					return err
				}

				return healthcheck.CheckPodsRunning(pods, "")

			}))

	checkers = append(checkers,
		*healthcheck.NewChecker("SMI extension proxies are healthy").
			WithHintAnchor("l5d-smi-proxy-healthy").
			Fatal().
			WithRetryDeadline(hc.RetryDeadline).
			SurfaceErrorOnRetry().
			WithCheck(func(ctx context.Context) error {
				return hc.CheckProxyHealth(ctx, hc.ControlPlaneNamespace, hc.DataPlaneNamespace)
			}))

	return healthcheck.NewCategory(linkerdSMIExtensionCheck, checkers, true)
}

func newCheckOptions() *checkOptions {
	return &checkOptions{
		wait:   300 * time.Second,
		output: healthcheck.TableOutput,
	}
}

func (options *checkOptions) validate() error {
	if options.output != healthcheck.TableOutput && options.output != healthcheck.JSONOutput {
		return fmt.Errorf("Invalid output type '%s'. Supported output types are: %s, %s", options.output, healthcheck.JSONOutput, healthcheck.TableOutput)
	}
	return nil
}

// newCmdCheck generates a new cobra command for the SMI extension.
func newCmdCheck() *cobra.Command {
	options := newCheckOptions()
	cmd := &cobra.Command{
		Use:   "check [flags]",
		Args:  cobra.NoArgs,
		Short: "Check the SMI extension for potential problems",
		Long: `Check the SMI extension for potential problems.

The check command will perform a series of checks to validate that the SMI
extension is configured correctly. If the command encounters a failure it will
print additional information about the failure and exit with a non-zero exit
code.`,
		Example: `  # Check that the SMI extension is up and running
  linkerd smi check`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return configureAndRunChecks(stdout, stderr, options)
		},
	}

	cmd.Flags().StringVarP(&options.output, "output", "o", options.output, "Output format. One of: basic, json")
	cmd.Flags().DurationVar(&options.wait, "wait", options.wait, "Maximum allowed time for all tests to pass")
	cmd.Flags().BoolVar(&options.proxy, "proxy", options.proxy, "Also run data-plane checks, to determine if the data plane is healthy")
	cmd.Flags().StringVarP(&options.namespace, "namespace", "n", options.namespace, "Namespace to use for --proxy checks (default: all namespaces)")
	cmd.Flags().StringVar(&options.pre, "pre", options.namespace, "Only run pre-installation checks, to determine if the extension can be installed")

	// stop marking these flags as hidden, once they are being supported
	cmd.Flags().MarkHidden("pre")
	cmd.Flags().MarkHidden("proxy")
	return cmd
}

func configureAndRunChecks(wout io.Writer, werr io.Writer, options *checkOptions) error {
	err := options.validate()
	if err != nil {
		return fmt.Errorf("Validation error when executing check command: %v", err)
	}

	hc := healthcheck.NewHealthChecker([]healthcheck.CategoryID{}, &healthcheck.Options{
		ControlPlaneNamespace: controlPlaneNamespace,
		KubeConfig:            kubeconfigPath,
		KubeContext:           kubeContext,
		Impersonate:           impersonate,
		ImpersonateGroup:      impersonateGroup,
		APIAddr:               apiAddr,
		RetryDeadline:         time.Now().Add(options.wait),
		DataPlaneNamespace:    options.namespace,
	})

	err = hc.InitializeKubeAPIClient()
	if err != nil {
		err = fmt.Errorf("Error initializing k8s API client: %s", err)
		fmt.Fprintln(werr, err)
		os.Exit(1)
	}

	hc.AppendCategories(smiCategory(hc))

	success, warning := healthcheck.RunChecks(wout, werr, hc, options.output)
	healthcheck.PrintChecksResult(wout, options.output, success, warning)

	if !success {
		os.Exit(1)
	}

	return nil
}
