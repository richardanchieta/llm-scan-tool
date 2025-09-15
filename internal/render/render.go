// Package render transforma o Summary em artefatos (Markdown e JSON).
package render

import (
	"bytes"
	//"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/richardanchieta/llm-scan-tool/internal/collect"
)

// BuildArtifacts recebe um Summary e retorna o Markdown e o JSON prontos.
func BuildArtifacts(sum *collect.Summary) (markdown string, jsonBytes []byte, err error) {
	j, err := sum.MarshalJSON()
	if err != nil {
		return "", nil, err
	}
	var b bytes.Buffer

	// YAML-like frontmatter
	fmt.Fprintf(&b, "---\n")
	fmt.Fprintf(&b, "generated_at: %s\n", sum.GeneratedAt.Format(time.RFC3339))
	fmt.Fprintf(&b, "root: %s\n", sum.Root)
	fmt.Fprintf(&b, "go_modules: %d\n", len(sum.GoModules))
	fmt.Fprintf(&b, "proto_files: %d\n", len(sum.Proto))
	fmt.Fprintf(&b, "sql_migrations: %d\n", len(sum.SQLMigrations))
	fmt.Fprintf(&b, "decisions: %d\n", len(sum.Decisions))
	fmt.Fprintf(&b, "---\n\n")

	b.WriteString("# Monorepo Snapshot (Optimized for LLM)\n\n")
	b.WriteString("_This file is an automatically generated, condensed map of the repository meant for LLM context seeding. It avoids large binaries, keeps only the top of key files, and surfaces decisions & APIs._\n\n")

	// Quick inventory
	b.WriteString("## Inventory\n\n")
	b.WriteString("| Item | Count |\n|---|---:|\n")
	b.WriteString(fmt.Sprintf("| Go modules | %d |\n", len(sum.GoModules)))
	b.WriteString(fmt.Sprintf("| Proto files | %d |\n", len(sum.Proto)))
	b.WriteString(fmt.Sprintf("| Make targets | %d |\n", len(sum.MakeTargets)))
	b.WriteString(fmt.Sprintf("| Dockerfiles | %d |\n", len(sum.Dockerfiles)))
	b.WriteString(fmt.Sprintf("| SQL migrations | %d |\n", len(sum.SQLMigrations)))
	b.WriteString(fmt.Sprintf("| ADR/Decisions | %d |\n", len(sum.Decisions)))
	b.WriteString(fmt.Sprintf("| README files | %d |\n", len(sum.Readmes)))
	b.WriteString("\n")

	// Tree (pruned)
	b.WriteString("## Repository Tree (pruned)\n\n```\n")
	for _, line := range sum.Tree {
		b.WriteString(line + "\n")
	}
	b.WriteString("```\n\n")

	// Test Coverage (Go + BDD)
	if sum.TestCoverage != nil {
		b.WriteString("## Test Coverage\n\n")

		// Go coverage
		if sum.TestCoverage.HasGoProfile {
			b.WriteString(fmt.Sprintf("- **Go coverage:** %.2f%%  (`%d/%d` statements)\n",
				sum.TestCoverage.Percent, sum.TestCoverage.CoveredStmts, sum.TestCoverage.TotalStmts))
		} else {
			b.WriteString("- **Go coverage:** (no coverprofile found)\n")
		}

		// BDD coverage / signals
		if sum.TestCoverage.BDD.FeatureFiles > 0 || len(sum.TestCoverage.BDD.Reports) > 0 {
			b.WriteString("  - **BDD (.feature):**\n")
			b.WriteString(fmt.Sprintf("    - feature files: **%d**\n", sum.TestCoverage.BDD.FeatureFiles))
			if sum.TestCoverage.BDD.Features+sum.TestCoverage.BDD.Scenarios+sum.TestCoverage.BDD.Steps > 0 {
				b.WriteString(fmt.Sprintf("    - cucumber totals: features=%d, scenarios=%d, steps=%d\n",
					sum.TestCoverage.BDD.Features, sum.TestCoverage.BDD.Scenarios, sum.TestCoverage.BDD.Steps))
			}
			if len(sum.TestCoverage.BDD.Reports) > 0 {
				// limitar listagem para não inflar contexto
				lim := sum.TestCoverage.BDD.Reports
				if len(lim) > 8 {
					lim = append(lim[:8], "…")
				}
				b.WriteString("    - reports: " + strings.Join(lim, ", ") + "\n")
			}
		}
		b.WriteString("\n")
	}

	// Go modules
	if len(sum.GoModules) > 0 {
		b.WriteString("## Go Modules\n\n")
		for _, m := range sum.GoModules {
			b.WriteString(fmt.Sprintf("- `%s` — **module**: `%s`\n", m.Path, strings.TrimSpace(m.Module)))
			if len(m.Requires) > 0 {
				uniq := uniqueSorted(m.Requires)
				if len(uniq) > 12 {
					uniq = uniq[:12]
					uniq = append(uniq, "…")
				}
				b.WriteString("  - deps: " + strings.Join(uniq, ", ") + "\n")
			}
		}
		b.WriteString("\n")
	}

	// Proto summary
	if len(sum.Proto) > 0 {
		b.WriteString("## Protobuf APIs\n\n")
		for _, p := range sum.Proto {
			b.WriteString(fmt.Sprintf("- `%s` — package: `%s`\n", p.File, p.Package))
			if len(p.Services) > 0 {
				b.WriteString("  - services: " + strings.Join(p.Services, ", ") + "\n")
			}
			if len(p.RPCs) > 0 {
				x := p.RPCs
				if len(x) > 20 {
					x = append(x[:20], "…")
				}
				b.WriteString("  - rpcs: " + strings.Join(x, ", ") + "\n")
			}
		}
		b.WriteString("\n")
	}

	// Make targets
	if len(sum.MakeTargets) > 0 {
		b.WriteString("## Make Targets (top-level)\n\n")
		uniq := uniqueSorted(sum.MakeTargets)
		if len(uniq) > 60 {
			uniq = append(uniq[:60], "…")
		}
		for _, t := range uniq {
			b.WriteString("- " + t + "\n")
		}
		b.WriteString("\n")
	}

	// SQL migrations and Dockerfiles
	if len(sum.SQLMigrations) > 0 || len(sum.Dockerfiles) > 0 {
		b.WriteString("## Build & Database Artifacts\n\n")
		if len(sum.Dockerfiles) > 0 {
			b.WriteString("**Dockerfiles**\n\n")
			for _, d := range sum.Dockerfiles {
				b.WriteString("- " + d + "\n")
			}
			b.WriteString("\n")
		}
		if len(sum.SQLMigrations) > 0 {
			b.WriteString("**SQL Migrations**\n\n")
			for _, s := range sum.SQLMigrations {
				b.WriteString("- " + s + "\n")
			}
			b.WriteString("\n")
		}
	}

	// Decisions
	if len(sum.Decisions) > 0 {
		b.WriteString("## Architecture Decisions (ADRs)\n\n")
		for _, d := range sum.Decisions {
			title := d.Title
			if title == "" {
				title = "(no title)"
			}
			s := d.Summary
			if len(s) > 240 {
				s = s[:240] + "…"
			}
			b.WriteString(fmt.Sprintf("- `%s` — **%s**\n  - %s\n", d.File, title, s))
		}
		b.WriteString("\n")
	}

	// Readmes and configs
	if len(sum.Readmes) > 0 {
		b.WriteString("## READMEs\n\n")
		for _, r := range sum.Readmes {
			b.WriteString("- " + r + "\n")
		}
		b.WriteString("\n")
	}
	if len(sum.EnvExamples) > 0 || len(sum.Licenses) > 0 {
		b.WriteString("## Misc\n\n")
		if len(sum.EnvExamples) > 0 {
			b.WriteString("**Env examples**\n\n")
			for _, e := range sum.EnvExamples {
				b.WriteString("- " + e + "\n")
			}
			b.WriteString("\n")
		}
		if len(sum.Licenses) > 0 {
			b.WriteString("**Licenses**\n\n")
			for _, l := range sum.Licenses {
				b.WriteString("- " + l + "\n")
			}
			b.WriteString("\n")
		}
	}

	// README Summaries
	if len(sum.ReadmeSummaries) > 0 {
		b.WriteString("## README Summaries\n\n")
		// ordenar pela chave do mapa (caminho) para saída estável
		var keys []string
		for k := range sum.ReadmeSummaries {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			rs := sum.ReadmeSummaries[k]
			title := rs.Title
			if title == "" {
				title = "(no title)"
			}
			b.WriteString(fmt.Sprintf("### %s\n", k))
			b.WriteString(fmt.Sprintf("- **Title:** %s\n", title))
			if rs.Objective != "" {
				b.WriteString(fmt.Sprintf("- **Objective:** %s\n", rs.Objective))
			}
			if rs.FirstPara != "" {
				b.WriteString(fmt.Sprintf("- **Summary:** %s\n", rs.FirstPara))
			}
			b.WriteString("\n")
		}
	}

	// Tech stats
	if len(sum.TechStats) > 0 {
		b.WriteString("## File Type Stats\n\n")
		type kv struct {
			K string
			V int
		}
		var arr []kv
		for k, v := range sum.TechStats {
			arr = append(arr, kv{k, v})
		}
		sort.Slice(arr, func(i, j int) bool { return arr[i].V > arr[j].V })
		b.WriteString("| Ext | Files |\n|---|---:|\n")
		limit := arr
		if len(limit) > 30 {
			limit = limit[:30]
		}
		for _, it := range limit {
			if it.K == "" {
				it.K = "(none)"
			}
			b.WriteString(fmt.Sprintf("| %s | %d |\n", it.K, it.V))
		}
		b.WriteString("\n")
	}

	// Footer
	b.WriteString("> Generated by `llm-scan-tool`. Safe to commit; intended for AI context windows.\n")

	return b.String(), j, nil
}

func uniqueSorted(in []string) []string {
	m := map[string]struct{}{}
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		m[s] = struct{}{}
	}
	var out []string
	for s := range m {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
