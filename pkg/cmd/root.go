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
	"github.com/spf13/cobra"
)

var (
	// Version is set during build
	Version = "dev"
)

// NewRootCommand creates the root command for kubectl-fluid
func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubectl-fluid",
		Short: "A kubectl plugin for inspecting Fluid Datasets and Runtimes",
		Long: `kubectl-fluid is a CLI tool for inspecting the status of Fluid Datasets
and their underlying Kubernetes resources in one unified view.

This plugin provides read-only operations to help you understand the current
state of your Fluid datasets, identify issues, and troubleshoot problems.

Usage:
  kubectl fluid inspect dataset <name> [-n namespace]

Examples:
  # Inspect a dataset in the default namespace
  kubectl fluid inspect dataset demo-data

  # Inspect a dataset in a specific namespace
  kubectl fluid inspect dataset demo-data -n fluid-system`,
		Version: Version,
	}

	// Add inspect subcommand
	cmd.AddCommand(NewInspectCommand())

	return cmd
}
