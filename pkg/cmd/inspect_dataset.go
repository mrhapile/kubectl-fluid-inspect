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
	"fmt"
	"os"

	"github.com/mrhapile/kubectl-fluid-inspect/pkg/inspect"
	"github.com/mrhapile/kubectl-fluid-inspect/pkg/k8s"
	"github.com/mrhapile/kubectl-fluid-inspect/pkg/output"
	"github.com/spf13/cobra"
)

type inspectDatasetOptions struct {
	namespace  string
	kubeconfig string
}

// NewInspectDatasetCommand creates the 'inspect dataset' subcommand
func NewInspectDatasetCommand() *cobra.Command {
	opts := &inspectDatasetOptions{}

	cmd := &cobra.Command{
		Use:   "dataset <name>",
		Short: "Inspect a Fluid Dataset and its underlying resources",
		Long: `Inspect a Fluid Dataset to view its current status, bound runtime,
and all underlying Kubernetes resources in a unified view.

The command fetches:
- Dataset CR status (phase, conditions)
- Bound Runtime CR status (master, worker, fuse phases)
- StatefulSets (master, worker)
- DaemonSet (fuse)
- PersistentVolumeClaim

Output includes ready/desired counts and highlights any issues.`,
		Example: `  # Inspect a dataset named "demo-data" in the default namespace
  kubectl fluid inspect dataset demo-data

  # Inspect a dataset in a specific namespace
  kubectl fluid inspect dataset demo-data -n fluid-system

  # Inspect using a specific kubeconfig
  kubectl fluid inspect dataset demo-data --kubeconfig ~/.kube/custom-config`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInspectDataset(args[0], opts)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&opts.namespace, "namespace", "n", "default", "The namespace of the dataset")
	cmd.Flags().StringVar(&opts.kubeconfig, "kubeconfig", "", "Path to the kubeconfig file (defaults to $KUBECONFIG or ~/.kube/config)")

	return cmd
}

func runInspectDataset(name string, opts *inspectDatasetOptions) error {
	// Create Kubernetes client
	client, err := k8s.NewClient(opts.kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Create inspector and run inspection
	inspector := inspect.NewDatasetInspector(client)
	result, err := inspector.Inspect(opts.namespace, name)
	if err != nil {
		return fmt.Errorf("failed to inspect dataset: %w", err)
	}

	// Output the result
	printer := output.NewTextPrinter(os.Stdout)
	printer.Print(result)

	return nil
}
