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

package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/mrhapile/kubectl-fluid-inspect/pkg/types"
)

// TextPrinter prints inspection results in human-readable text format
type TextPrinter struct {
	writer io.Writer
}

// NewTextPrinter creates a new TextPrinter
func NewTextPrinter(w io.Writer) *TextPrinter {
	return &TextPrinter{writer: w}
}

// Print prints the inspection result
func (p *TextPrinter) Print(result *types.InspectionResult) {
	p.printHeader()
	p.printDatasetInfo(&result.Dataset)

	if result.Runtime != nil {
		p.printRuntimeInfo(result.Runtime)
	}

	p.printResourceStatus(&result.Resources)
	p.printConditions(result.Dataset.Conditions)
	p.printFooter()
}

func (p *TextPrinter) printHeader() {
	p.println(strings.Repeat("=", 80))
	p.println("                          FLUID DATASET INSPECTION")
	p.println(strings.Repeat("=", 80))
	p.println("")
}

func (p *TextPrinter) printFooter() {
	p.println(strings.Repeat("=", 80))
}

func (p *TextPrinter) printDatasetInfo(info *types.DatasetInfo) {
	p.printf("DATASET: %s\n", info.Name)
	p.printf("NAMESPACE: %s\n", info.Namespace)
	p.printf("STATUS: %s\n", p.formatPhase(info.Phase))

	if info.UfsTotal != "" {
		p.printf("UFS TOTAL: %s\n", info.UfsTotal)
	}

	if info.FileNum != "" {
		p.printf("FILE COUNT: %s\n", info.FileNum)
	}

	if len(info.MountPoints) > 0 {
		p.println("")
		p.println("MOUNT POINTS:")
		for _, mp := range info.MountPoints {
			p.printf("  - %s\n", mp)
		}
	}

	p.println("")
}

func (p *TextPrinter) printRuntimeInfo(info *types.RuntimeInfo) {
	p.println(strings.Repeat("=", 80))
	p.printf("RUNTIME: %s (%s)\n", info.Name, strings.Title(info.Type)+"Runtime")
	p.println(strings.Repeat("=", 80))
	p.println("")

	p.println("COMPONENT STATUS:")
	p.println(strings.Repeat("-", 40))

	// Master status
	masterStatus := p.formatComponentStatus("Master StatefulSet", info.Master)
	p.println(masterStatus)

	// Worker status
	workerStatus := p.formatComponentStatus("Worker StatefulSet", info.Worker)
	p.println(workerStatus)

	// Fuse status
	fuseStatus := p.formatFuseStatus("Fuse DaemonSet", info.Fuse)
	p.println(fuseStatus)

	p.println("")
}

func (p *TextPrinter) printResourceStatus(resources *types.ResourceStatus) {
	p.println("KUBERNETES RESOURCES:")
	p.println(strings.Repeat("-", 40))

	// Master StatefulSet
	if resources.MasterStatefulSet != nil {
		sts := resources.MasterStatefulSet
		status := fmt.Sprintf("Ready (%d/%d)", sts.ReadyReplicas, sts.Replicas)
		indicator := ""
		if !sts.Healthy {
			indicator = " ⚠️"
			status = fmt.Sprintf("Not Ready (%d/%d)", sts.ReadyReplicas, sts.Replicas)
		}
		p.printf("Master StatefulSet: %s%s\n", status, indicator)
	} else {
		p.println("Master StatefulSet: Not Found")
	}

	// Worker StatefulSet
	if resources.WorkerStatefulSet != nil {
		sts := resources.WorkerStatefulSet
		status := fmt.Sprintf("Ready (%d/%d)", sts.ReadyReplicas, sts.Replicas)
		indicator := ""
		if !sts.Healthy {
			indicator = " ⚠️"
			status = fmt.Sprintf("Not Ready (%d/%d)", sts.ReadyReplicas, sts.Replicas)
		}
		p.printf("Worker StatefulSet: %s%s\n", status, indicator)
	} else {
		p.println("Worker StatefulSet: Not Found")
	}

	// Fuse DaemonSet
	if resources.FuseDaemonSet != nil {
		ds := resources.FuseDaemonSet
		status := fmt.Sprintf("Ready (%d/%d)", ds.Ready, ds.DesiredScheduled)
		indicator := ""
		if !ds.Healthy {
			indicator = " ⚠️"
		}
		p.printf("Fuse DaemonSet:     %s%s\n", status, indicator)
	} else {
		p.println("Fuse DaemonSet:     Not Found")
	}

	// PVC
	if resources.PVC != nil {
		pvc := resources.PVC
		p.printf("PVC:                %s", pvc.Phase)
		if pvc.VolumeName != "" {
			p.printf(" (%s)", pvc.VolumeName)
		}
		if pvc.Phase != "Bound" {
			p.print(" ⚠️")
		}
		p.println("")
	} else {
		p.println("PVC:                Not Found")
	}

	p.println("")
}

func (p *TextPrinter) printConditions(conditions []types.ConditionInfo) {
	if len(conditions) == 0 {
		return
	}

	p.println(strings.Repeat("=", 80))
	p.println("CONDITIONS:")
	p.println(strings.Repeat("=", 80))

	for _, cond := range conditions {
		statusSymbol := "❌"
		if cond.Status == "True" {
			statusSymbol = "✓"
		}

		p.printf("%s %s: %s", statusSymbol, cond.Type, cond.Status)
		if cond.Reason != "" {
			p.printf(" (%s)", cond.Reason)
		}
		p.println("")

		if cond.Message != "" {
			p.printf("   %s\n", cond.Message)
		}
	}

	p.println("")
}

func (p *TextPrinter) formatPhase(phase string) string {
	switch phase {
	case "Bound":
		return "Bound ✓"
	case "Pending":
		return "Pending ⏳"
	case "Failed":
		return "Failed ❌"
	case "NotBound":
		return "NotBound ⚠️"
	default:
		return phase
	}
}

func (p *TextPrinter) formatComponentStatus(name string, status types.ComponentStatus) string {
	if status.DesiredScheduled == 0 && status.Ready == 0 {
		return fmt.Sprintf("%-20s Not Found", name+":")
	}

	indicator := ""
	if status.Ready < status.DesiredScheduled {
		indicator = " ⚠️"
	}

	phase := status.Phase
	if phase == "" {
		phase = "Unknown"
	}

	return fmt.Sprintf("%-20s %s (%d/%d)%s", name+":", phase, status.Ready, status.DesiredScheduled, indicator)
}

func (p *TextPrinter) formatFuseStatus(name string, status types.ComponentStatus) string {
	if status.DesiredScheduled == 0 && status.Ready == 0 {
		return fmt.Sprintf("%-20s Not Found", name+":")
	}

	indicator := ""
	if status.Ready < status.DesiredScheduled {
		indicator = " ⚠️"
	}

	phase := status.Phase
	if phase == "" {
		phase = "Unknown"
	}

	return fmt.Sprintf("%-20s %s (%d/%d)%s", name+":", phase, status.Ready, status.DesiredScheduled, indicator)
}

func (p *TextPrinter) println(s string) {
	fmt.Fprintln(p.writer, s)
}

func (p *TextPrinter) printf(format string, args ...interface{}) {
	fmt.Fprintf(p.writer, format, args...)
}

func (p *TextPrinter) print(s string) {
	fmt.Fprint(p.writer, s)
}
