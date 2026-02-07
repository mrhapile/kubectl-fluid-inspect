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
		Short: "A kubectl plugin for inspecting and diagnosing Fluid Datasets and Runtimes",
		Long: `kubectl-fluid is a CLI tool for inspecting and diagnosing the status of Fluid Datasets
and their underlying Kubernetes resources in one unified view.

This plugin provides read-only operations to help you understand the current
state of your Fluid datasets, identify issues, and troubleshoot problems.

Commands:
  inspect  - Quick status overview of a Dataset and Runtime
  diagnose - Comprehensive debugging with logs, events, and failure analysis

Examples:
  # Quick inspect of a dataset
  kubectl fluid inspect dataset demo-data

  # Comprehensive diagnosis with logs and events
  kubectl fluid diagnose dataset demo-data

  # Generate diagnostic archive for sharing
  kubectl fluid diagnose dataset demo-data --archive

  # Export AI-ready diagnostic context
  kubectl fluid diagnose dataset demo-data --output json`,
		Version: Version,
	}

	// Add subcommands
	cmd.AddCommand(NewInspectCommand())
	cmd.AddCommand(NewDiagnoseCommand())

	return cmd
}
