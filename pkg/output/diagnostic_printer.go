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

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

// DiagnosticPrinter prints diagnostic results with colors and formatting
type DiagnosticPrinter struct {
	writer   io.Writer
	useColor bool
}

// NewDiagnosticPrinter creates a new DiagnosticPrinter
func NewDiagnosticPrinter(w io.Writer) *DiagnosticPrinter {
	return &DiagnosticPrinter{
		writer:   w,
		useColor: true, // Can be made configurable
	}
}

// Print prints the diagnostic result
func (p *DiagnosticPrinter) Print(result *types.DiagnosticResult) {
	p.printHeader(result)
	p.printResourceTree(result)
	p.println("")
	p.printFailureHints(result)
	p.printEvents(result)
	p.printLogs(result)
	p.printFooter(result)
}

func (p *DiagnosticPrinter) printHeader(result *types.DiagnosticResult) {
	p.println("")
	p.println(p.color(colorBold, "╔══════════════════════════════════════════════════════════════════════════════╗"))
	p.println(p.color(colorBold, "║                      FLUID DATASET DIAGNOSTIC REPORT                        ║"))
	p.println(p.color(colorBold, "╚══════════════════════════════════════════════════════════════════════════════╝"))
	p.println("")

	// Dataset info
	p.printf("  %s: %s\n", p.color(colorBold, "Dataset"), result.DatasetName)
	p.printf("  %s: %s\n", p.color(colorBold, "Namespace"), result.Namespace)
	p.printf("  %s: %s\n", p.color(colorBold, "Collected At"), result.CollectedAt.Format("2006-01-02 15:04:05"))
	p.printf("  %s: %s\n", p.color(colorBold, "Health Status"), p.formatHealthStatus(result.HealthStatus))
	p.println("")
}

func (p *DiagnosticPrinter) printResourceTree(result *types.DiagnosticResult) {
	phase := extractPhaseFromDiagnostic(result)

	p.println(p.color(colorBold, "=== RESOURCE HIERARCHY ==="))
	p.println("")

	// Dataset node
	p.printf("  %s %s [%s]\n",
		p.color(colorCyan, "📦"),
		p.color(colorBold, "Dataset: "+result.DatasetName),
		p.formatPhaseColor(phase))

	// Runtime node
	if result.RuntimeType != "" {
		runtimeName := strings.TrimSuffix(result.RuntimeType, "s")
		p.printf("  %s\n", p.color(colorDim, "│"))
		p.printf("  └── %s %s: %s\n",
			p.color(colorCyan, "⚙️"),
			p.color(colorBold, "Runtime"),
			runtimeName)

		// Component nodes
		if result.Resources.Master != nil {
			p.printComponentNode("Master", result.Resources.Master, false)
		}
		if result.Resources.Workers != nil {
			p.printComponentNode("Workers", result.Resources.Workers, false)
		}
		if result.Resources.Fuse != nil {
			p.printComponentNode("Fuse", result.Resources.Fuse, true)
		}
	}

	// PVC node
	if result.Resources.PVC != nil {
		pvc := result.Resources.PVC
		pvcStatus := p.formatStatusIcon(pvc.Phase == "Bound")
		p.printf("  %s\n", p.color(colorDim, "│"))
		p.printf("  └── %s PVC: %s %s\n",
			p.color(colorCyan, "💾"),
			pvc.Name,
			pvcStatus)
		if pvc.VolumeName != "" {
			p.printf("      └── PV: %s\n", pvc.VolumeName)
		}
	}

	p.println("")
}

func (p *DiagnosticPrinter) printComponentNode(name string, status *types.PodGroupStatus, isLast bool) {
	prefix := "├──"
	childPrefix := "│   "
	if isLast {
		prefix = "└──"
		childPrefix = "    "
	}

	statusIcon := p.formatStatusIcon(status.Healthy)
	readyStr := fmt.Sprintf("%d/%d", status.Ready, status.Desired)

	p.printf("      %s %s %s: %s %s\n",
		prefix,
		p.color(colorCyan, "📊"),
		name,
		readyStr,
		statusIcon)

	// Print failing pods if any
	if len(status.FailingPods) > 0 {
		for i, pod := range status.FailingPods {
			podPrefix := "├──"
			if i == len(status.FailingPods)-1 {
				podPrefix = "└──"
			}
			p.printf("      %s   %s %s %s: %s\n",
				childPrefix,
				podPrefix,
				p.color(colorRed, "⚠️"),
				pod.Name,
				p.color(colorRed, pod.Reason))
		}
	}
}

func (p *DiagnosticPrinter) printFailureHints(result *types.DiagnosticResult) {
	if len(result.FailureHints) == 0 {
		p.println(p.color(colorGreen, "=== NO ISSUES DETECTED ==="))
		p.println("")
		return
	}

	p.println(p.color(colorBold, "=== DETECTED ISSUES ==="))
	p.println("")

	// Group by severity
	criticals := []types.FailureHint{}
	warnings := []types.FailureHint{}
	infos := []types.FailureHint{}

	for _, hint := range result.FailureHints {
		switch hint.Severity {
		case "critical":
			criticals = append(criticals, hint)
		case "warning":
			warnings = append(warnings, hint)
		default:
			infos = append(infos, hint)
		}
	}

	printHints := func(hints []types.FailureHint, icon, colorCode string) {
		for _, hint := range hints {
			p.printf("  %s %s [%s]\n",
				p.color(colorCode, icon),
				p.color(colorCode, hint.Issue),
				hint.Component)
			p.printf("     %s %s\n",
				p.color(colorDim, "→"),
				hint.Suggestion)
			if hint.Evidence != "" {
				p.printf("     %s %s\n",
					p.color(colorDim, "Evidence:"),
					p.truncate(hint.Evidence, 80))
			}
			p.println("")
		}
	}

	if len(criticals) > 0 {
		p.println(p.color(colorRed, "  CRITICAL:"))
		printHints(criticals, "❌", colorRed)
	}
	if len(warnings) > 0 {
		p.println(p.color(colorYellow, "  WARNINGS:"))
		printHints(warnings, "⚠️", colorYellow)
	}
	if len(infos) > 0 {
		p.println(p.color(colorBlue, "  INFO:"))
		printHints(infos, "ℹ️", colorBlue)
	}
}

func (p *DiagnosticPrinter) printEvents(result *types.DiagnosticResult) {
	if len(result.Events) == 0 {
		return
	}

	p.println(p.color(colorBold, "=== RECENT EVENTS ==="))
	p.println("")

	// Print last 10 events
	count := 10
	if len(result.Events) < count {
		count = len(result.Events)
	}

	// Table header
	p.printf("  %-12s %-20s %-15s %s\n",
		p.color(colorBold, "TYPE"),
		p.color(colorBold, "OBJECT"),
		p.color(colorBold, "REASON"),
		p.color(colorBold, "MESSAGE"))
	p.println("  " + strings.Repeat("-", 76))

	for i := 0; i < count; i++ {
		event := result.Events[i]
		typeColor := colorGreen
		if event.Type == "Warning" {
			typeColor = colorYellow
		}

		p.printf("  %-12s %-20s %-15s %s\n",
			p.color(typeColor, event.Type),
			p.truncate(event.ObjectName, 20),
			p.truncate(event.Reason, 15),
			p.truncate(event.Message, 40))
	}

	if len(result.Events) > count {
		p.printf("  %s\n", p.color(colorDim, fmt.Sprintf("... and %d more events", len(result.Events)-count)))
	}
	p.println("")
}

func (p *DiagnosticPrinter) printLogs(result *types.DiagnosticResult) {
	hasLogs := false

	if result.Logs.Master != nil && result.Logs.Master.Logs != "" {
		hasLogs = true
	}
	if len(result.Logs.Workers) > 0 {
		for _, w := range result.Logs.Workers {
			if w.Logs != "" {
				hasLogs = true
				break
			}
		}
	}
	if len(result.Logs.Fuse) > 0 {
		for _, f := range result.Logs.Fuse {
			if f.Logs != "" {
				hasLogs = true
				break
			}
		}
	}

	if !hasLogs {
		return
	}

	p.println(p.color(colorBold, "=== LOGS (TAIL) ==="))
	p.println("")

	if result.Logs.Master != nil && result.Logs.Master.Logs != "" {
		p.printLogSection("MASTER", result.Logs.Master)
	}

	for i, entry := range result.Logs.Workers {
		if entry.Logs != "" {
			label := fmt.Sprintf("WORKER-%d", i)
			p.printLogSection(label, &entry)
		}
	}

	for i, entry := range result.Logs.Fuse {
		if entry.Logs != "" {
			label := fmt.Sprintf("FUSE-%d (FAILING)", i)
			p.printLogSection(label, &entry)
		}
	}
}

func (p *DiagnosticPrinter) printLogSection(label string, entry *types.LogEntry) {
	p.printf("  ┌─ %s [%s/%s] (%d lines)\n",
		p.color(colorCyan, label),
		entry.PodName,
		entry.ContainerName,
		entry.TailLines)
	p.println("  │")

	// Print last N lines
	lines := strings.Split(entry.Logs, "\n")
	maxLines := 15
	start := 0
	if len(lines) > maxLines {
		start = len(lines) - maxLines
		p.printf("  │ %s\n", p.color(colorDim, fmt.Sprintf("... %d lines truncated ...", start)))
	}

	for i := start; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) != "" {
			// Highlight error keywords
			if strings.Contains(strings.ToLower(line), "error") ||
				strings.Contains(strings.ToLower(line), "exception") ||
				strings.Contains(strings.ToLower(line), "failed") {
				p.printf("  │ %s\n", p.color(colorRed, p.truncate(line, 75)))
			} else if strings.Contains(strings.ToLower(line), "warn") {
				p.printf("  │ %s\n", p.color(colorYellow, p.truncate(line, 75)))
			} else {
				p.printf("  │ %s\n", p.truncate(line, 75))
			}
		}
	}

	p.println("  └─")
	p.println("")
}

func (p *DiagnosticPrinter) printFooter(result *types.DiagnosticResult) {
	p.println(p.color(colorBold, "═══════════════════════════════════════════════════════════════════════════════"))
	p.printf("  %s Use '%s' for machine-readable output\n",
		p.color(colorDim, "TIP:"),
		p.color(colorCyan, "kubectl fluid diagnose dataset "+result.DatasetName+" --output json"))
	p.printf("  %s Use '%s' to generate shareable archive\n",
		p.color(colorDim, "TIP:"),
		p.color(colorCyan, "kubectl fluid diagnose dataset "+result.DatasetName+" --archive"))
	p.println(p.color(colorBold, "═══════════════════════════════════════════════════════════════════════════════"))
	p.println("")
}

// Helper functions

func (p *DiagnosticPrinter) formatHealthStatus(status types.HealthStatus) string {
	switch status {
	case types.HealthStatusHealthy:
		return p.color(colorGreen, "✅ Healthy")
	case types.HealthStatusDegraded:
		return p.color(colorYellow, "⚠️  Degraded")
	case types.HealthStatusUnhealthy:
		return p.color(colorRed, "❌ Unhealthy")
	default:
		return p.color(colorDim, "❓ Unknown")
	}
}

func (p *DiagnosticPrinter) formatPhaseColor(phase string) string {
	switch phase {
	case "Bound":
		return p.color(colorGreen, "Bound ✓")
	case "Pending":
		return p.color(colorYellow, "Pending ⏳")
	case "Failed":
		return p.color(colorRed, "Failed ❌")
	case "NotBound":
		return p.color(colorYellow, "NotBound ⚠️")
	default:
		return p.color(colorDim, phase)
	}
}

func (p *DiagnosticPrinter) formatStatusIcon(healthy bool) string {
	if healthy {
		return p.color(colorGreen, "✓")
	}
	return p.color(colorRed, "✗")
}

func (p *DiagnosticPrinter) color(code, text string) string {
	if !p.useColor {
		return text
	}
	return code + text + colorReset
}

func (p *DiagnosticPrinter) truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func (p *DiagnosticPrinter) println(s string) {
	fmt.Fprintln(p.writer, s)
}

func (p *DiagnosticPrinter) printf(format string, args ...interface{}) {
	fmt.Fprintf(p.writer, format, args...)
}

func extractPhaseFromDiagnostic(result *types.DiagnosticResult) string {
	// Extract phase from YAML
	lines := strings.Split(result.DatasetYAML, "\n")
	for _, line := range lines {
		if strings.Contains(line, "phase:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "Unknown"
}
