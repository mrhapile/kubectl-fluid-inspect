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

// NewDiagnoseCommand creates the diagnose subcommand
func NewDiagnoseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diagnose",
		Short: "Diagnose Fluid resources and collect debugging information",
		Long: `Diagnose provides comprehensive debugging information about Fluid resources
by collecting CR snapshots, Kubernetes events, pod status, and logs into a
single, structured output.

This command automates the manual workflow of running multiple kubectl commands
(get, describe, logs) and correlating the results.`,
	}

	// Add dataset subcommand
	cmd.AddCommand(NewDiagnoseDatasetCommand())

	return cmd
}
