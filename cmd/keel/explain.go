package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/HueCodes/keel/internal/analyzer"
	"github.com/HueCodes/keel/internal/rules/bestpractice"
	"github.com/HueCodes/keel/internal/rules/performance"
	"github.com/HueCodes/keel/internal/rules/security"
	"github.com/HueCodes/keel/internal/rules/style"
)

type ruleInfo struct {
	ID          string
	Name        string
	Description string
	Category    analyzer.Category
	Severity    analyzer.Severity
}

func explainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "explain [rule]",
		Short: "Show detailed explanation of a rule",
		Long:  "Show detailed explanation of a rule or list all available rules if no argument is given.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Collect all rules
			rules := collectAllRules()

			if len(args) == 0 {
				// List all rules
				return listRules(rules)
			}

			// Find specific rule
			ruleID := strings.ToUpper(args[0])
			for _, r := range rules {
				if r.ID == ruleID {
					return explainRule(r)
				}
			}

			return fmt.Errorf("rule %q not found", args[0])
		},
	}

	return cmd
}

func collectAllRules() []ruleInfo {
	var rules []ruleInfo

	for _, r := range security.All() {
		rules = append(rules, ruleInfo{
			ID:          r.ID(),
			Name:        r.Name(),
			Description: r.Description(),
			Category:    r.Category(),
			Severity:    r.Severity(),
		})
	}
	for _, r := range performance.All() {
		rules = append(rules, ruleInfo{
			ID:          r.ID(),
			Name:        r.Name(),
			Description: r.Description(),
			Category:    r.Category(),
			Severity:    r.Severity(),
		})
	}
	for _, r := range bestpractice.All() {
		rules = append(rules, ruleInfo{
			ID:          r.ID(),
			Name:        r.Name(),
			Description: r.Description(),
			Category:    r.Category(),
			Severity:    r.Severity(),
		})
	}
	for _, r := range style.All() {
		rules = append(rules, ruleInfo{
			ID:          r.ID(),
			Name:        r.Name(),
			Description: r.Description(),
			Category:    r.Category(),
			Severity:    r.Severity(),
		})
	}

	sort.Slice(rules, func(i, j int) bool {
		return rules[i].ID < rules[j].ID
	})

	return rules
}

func listRules(rules []ruleInfo) error {
	fmt.Println("Available rules:")
	fmt.Println()

	categories := map[analyzer.Category][]ruleInfo{}
	for _, r := range rules {
		categories[r.Category] = append(categories[r.Category], r)
	}

	categoryOrder := []analyzer.Category{
		analyzer.CategorySecurity,
		analyzer.CategoryPerformance,
		analyzer.CategoryBestPractice,
		analyzer.CategoryStyle,
	}

	for _, cat := range categoryOrder {
		catRules := categories[cat]
		if len(catRules) == 0 {
			continue
		}

		fmt.Printf("## %s\n", strings.Title(string(cat)))
		for _, r := range catRules {
			fmt.Printf("  %s  %-30s  %s\n", r.ID, r.Name, severityIcon(r.Severity))
		}
		fmt.Println()
	}

	fmt.Printf("Total: %d rules\n", len(rules))
	fmt.Println()
	fmt.Println("Use 'keel explain <rule>' for detailed information about a specific rule.")

	return nil
}

func explainRule(r ruleInfo) error {
	fmt.Fprintf(os.Stdout, "Rule: %s (%s)\n", r.ID, r.Name)
	fmt.Fprintf(os.Stdout, "Category: %s\n", r.Category)
	fmt.Fprintf(os.Stdout, "Severity: %s %s\n", severityIcon(r.Severity), r.Severity)
	fmt.Println()
	fmt.Println("Description:")
	fmt.Printf("  %s\n", r.Description)
	fmt.Println()
	return nil
}

func severityIcon(s analyzer.Severity) string {
	switch s {
	case analyzer.SeverityError:
		return "error"
	case analyzer.SeverityWarning:
		return "warning"
	case analyzer.SeverityInfo:
		return "info"
	default:
		return "hint"
	}
}
