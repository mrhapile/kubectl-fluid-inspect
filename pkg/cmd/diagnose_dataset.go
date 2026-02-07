/*
Copyright 2026 kubectl-fluid-inspect Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/mrhapile/kubectl-fluid-inspect/pkg/diagnose"
	"github.com/mrhapile/kubectl-fluid-inspect/pkg/diagnose/mock"
	"github.com/mrhapile/kubectl-fluid-inspect/pkg/k8s"
	"github.com/mrhapile/kubectl-fluid-inspect/pkg/output"
	"github.com/mrhapile/kubectl-fluid-inspect/pkg/types"
	"github.com/spf13/cobra"
)

type diagnoseDatasetOptions struct {
	namespace  string
	kubeconfig string
	archive    bool
	outputFmt  string
	mockMode   bool
}

// NewDiagnoseDatasetCommand creates the 'diagnose dataset' subcommand
func NewDiagnoseDatasetCommand() *cobra.Command {
	opts := &diagnoseDatasetOptions{}

	cmd := &cobra.Command{
		Use:   "dataset <name>",
		Short: "Diagnose a Fluid Dataset and collect debugging information",
		Long: `Diagnose a Fluid Dataset by collecting comprehensive debugging information
including CR snapshots, Kubernetes events, pod status, and logs.

This command automates the manual workflow of running multiple kubectl commands
(get, describe, logs) and correlating the results.

The diagnostic pipeline collects:
  1. Dataset and Runtime CR snapshots (cleaned YAML)
  2. Kubernetes events related to the Dataset
  3. Runtime resource status (StatefulSets, DaemonSet, PVC)
  4. Container logs (master, worker, failing fuse pods)

The output includes automatic failure analysis with hints and suggestions.

MOCK MODE:
  Use --mock to run diagnose with simulated Fluid resources.
  No Kubernetes cluster is required. This is useful for:
  - Demos and documentation screenshots
  - Development and testing
  - Proposal proof-of-work`,
		Example: `  # Diagnose a dataset in the default namespace
  kubectl fluid diagnose dataset demo-data

  # Diagnose with JSON output (for AI integration)
  kubectl fluid diagnose dataset demo-data --output json

  # Generate a diagnostic archive for sharing
  kubectl fluid diagnose dataset demo-data --archive

  # Diagnose in a specific namespace
  kubectl fluid diagnose dataset demo-data -n fluid-system

  # Run with mock data (no cluster required)
  kubectl fluid diagnose dataset demo-data --mock

  # Generate mock archive for demos
  kubectl fluid diagnose dataset demo-data --mock --archive

  # Export mock JSON for AI testing
  kubectl fluid diagnose dataset demo-data --mock -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDiagnoseDataset(args[0], opts)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&opts.namespace, "namespace", "n", "default", "The namespace of the dataset")
	cmd.Flags().StringVar(&opts.kubeconfig, "kubeconfig", "", "Path to the kubeconfig file")
	cmd.Flags().BoolVar(&opts.archive, "archive", false, "Generate a diagnostic archive (.tar.gz)")
	cmd.Flags().StringVarP(&opts.outputFmt, "output", "o", "text", "Output format: text, json")
	cmd.Flags().BoolVar(&opts.mockMode, "mock", false, "Use mock data (no Kubernetes cluster required, for demos/development)")

	return cmd
}

func runDiagnoseDataset(name string, opts *diagnoseDatasetOptions) error {
	var result *types.DiagnosticResult
	var ctx *types.DiagnosticContext

	if opts.mockMode {
		// Mock mode: use simulated data, no K8s client needed
		result = mock.MockDiagnosticResult(name, opts.namespace)
		ctx = mock.MockDiagnosticContext(name, opts.namespace)
	} else {
		// Real mode: connect to Kubernetes
		client, err := k8s.NewClient(opts.kubeconfig)
		if err != nil {
			return fmt.Errorf("failed to create Kubernetes client: %w", err)
		}

		diagnoser := diagnose.NewDatasetDiagnoser(client)
		result, err = diagnoser.Diagnose(context.Background(), opts.namespace, name)
		if err != nil {
			return fmt.Errorf("failed to diagnose dataset: %w", err)
		}
		ctx = diagnoser.ToContext(result)
	}

	// Handle output based on flags
	if opts.archive {
		// Generate archive
		archiver := output.NewArchiver()
		archivePath, err := archiver.CreateArchive(result)
		if err != nil {
			return fmt.Errorf("failed to create archive: %w", err)
		}
		if opts.mockMode {
			fmt.Printf("✅ Mock diagnostic archive created: %s\n", archivePath)
		} else {
			fmt.Printf("✅ Diagnostic archive created: %s\n", archivePath)
		}
		return nil
	}

	switch opts.outputFmt {
	case "json":
		// Output AI-ready context as JSON
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(ctx)
	case "text":
		fallthrough
	default:
		// Human-readable output
		printer := output.NewDiagnosticPrinter(os.Stdout)
		printer.Print(result)
	}

	return nil
}
