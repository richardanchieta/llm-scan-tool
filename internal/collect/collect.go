// Package collect agrega metadados do repositório (módulos, proto, migrações,
// READMEs, decisões etc.) para geração do artefato otimizado para LLM.
package collect

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/richardanchieta/llm-scan-tool/internal/files"
)

// Config define parâmetros de varredura (root, filtros, limites, etc.).
type Config struct {
	Root            string
	MaxFileBytes    int64
	Threads         int
	IncludeGlobsCSV string
	ExcludeGlobsCSV string
	TreeDepth       int
}

// ReadmeSummary guarda um extrato leve de um README (título/objetivo/primeiro parágrafo).
type ReadmeSummary struct {
	File      string `json:"file"`
	Title     string `json:"title"`
	FirstPara string `json:"first_para"`
	Objective string `json:"objective"` // opcional: captura seção "Objetivo"/"Objective"
}

// Summary é o objeto principal agregado pelo coletor; base para render Markdown/JSON.
type Summary struct {
	Root            string                   `json:"root"`
	GeneratedAt     time.Time                `json:"generated_at"`
	GoModules       []GoModule               `json:"go_modules"`
	Proto           []ProtoInfo              `json:"proto"`
	MakeTargets     []string                 `json:"make_targets"`
	Dockerfiles     []string                 `json:"dockerfiles"`
	SQLMigrations   []string                 `json:"sql_migrations"`
	Decisions       []Decision               `json:"decisions"`
	EnvExamples     []string                 `json:"env_examples"`
	Licenses        []string                 `json:"licenses"`
	Readmes         []string                 `json:"readmes"`
	ReadmeSummaries map[string]ReadmeSummary `json:"readme_summaries"`
	TechStats       map[string]int           `json:"tech_stats"`
	Tree            []string                 `json:"tree"`
	NotableConfigs  []string                 `json:"notable_configs"`
}

// GoModule descreve um módulo Go encontrado (path/module/requires).
type GoModule struct {
	Path     string   `json:"path"`
	Module   string   `json:"module"`
	Requires []string `json:"requires"`
}

// ProtoInfo descreve um arquivo/projeto Protobuf (package, services, RPCs).
type ProtoInfo struct {
	File     string   `json:"file"`
	Package  string   `json:"package"`
	Services []string `json:"services"`
	RPCs     []string `json:"rpcs"`
}

// Decision representa uma ADR/decisão técnica detectada.
type Decision struct {
	File    string `json:"file"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

// Scan executa a varredura e devolve um *Summary pronto para renderização.
func Scan(ctx context.Context, cfg Config) (*Summary, error) {
	if cfg.Threads <= 0 {
		cfg.Threads = runtime.NumCPU()
	}
	sum := &Summary{
		Root:            cfg.Root,
		GeneratedAt:     time.Now(),
		TechStats:       map[string]int{},
		ReadmeSummaries: map[string]ReadmeSummary{},
	}
	includeGlobs := splitCSV(cfg.IncludeGlobsCSV)
	excludeGlobs := append(files.DefaultIgnore(), splitCSV(cfg.ExcludeGlobsCSV)...)

	// Walk
	var paths []string
	err := filepath.WalkDir(cfg.Root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}
		rel, _ := filepath.Rel(cfg.Root, path)
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		// Exclude dirs quick
		if d.IsDir() {
			if files.MatchAny(excludeGlobs, rel+"/") {
				return filepath.SkipDir
			}
			return nil
		}
		// Exclude files
		if files.MatchAny(excludeGlobs, rel) && !files.MatchAny(includeGlobs, rel) {
			return nil
		}
		paths = append(paths, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Concurrent process files
	sem := make(chan struct{}, cfg.Threads)
	var mu sync.Mutex
	var wg sync.WaitGroup
loop:
	for _, p := range paths {
		select {
		case <-ctx.Done():
			break loop
		default:
		}
		wg.Add(1)
		p := p
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			full := filepath.Join(cfg.Root, p)
			lower := strings.ToLower(p)

			switch {
			case strings.HasSuffix(lower, "go.mod"):
				if gm, err := parseGoMod(full); err == nil {
					mu.Lock()
					sum.GoModules = append(sum.GoModules, *gm)
					mu.Unlock()
				}
			case strings.HasSuffix(lower, ".proto"):
				if pi, err := parseProto(full, cfg.MaxFileBytes); err == nil {
					mu.Lock()
					sum.Proto = append(sum.Proto, *pi)
					mu.Unlock()
				}
			case filepath.Base(lower) == "makefile" || strings.HasSuffix(lower, ".mk"):
				if ts, err := parseMakeTargets(full, cfg.MaxFileBytes); err == nil {
					mu.Lock()
					sum.MakeTargets = append(sum.MakeTargets, ts...)
					mu.Unlock()
				}
			case strings.HasSuffix(lower, "dockerfile") || strings.HasPrefix(filepath.Base(lower), "dockerfile."):
				mu.Lock()
				sum.Dockerfiles = append(sum.Dockerfiles, p)
				mu.Unlock()
			case strings.HasSuffix(lower, ".sql"):
				if strings.Contains(lower, "migrat") || strings.Contains(lower, "schema") {
					mu.Lock()
					sum.SQLMigrations = append(sum.SQLMigrations, p)
					mu.Unlock()
				}
			case strings.HasSuffix(lower, ".md") && (strings.Contains(lower, "/docs/decisions/") || strings.Contains(lower, "/adr")):
				if dec, err := parseDecision(full, cfg.MaxFileBytes); err == nil {
					mu.Lock()
					sum.Decisions = append(sum.Decisions, *dec)
					mu.Unlock()
				}
			case strings.HasSuffix(lower, ".env") || strings.HasSuffix(lower, ".env.example") || strings.HasSuffix(lower, ".sample"):
				mu.Lock()
				sum.EnvExamples = append(sum.EnvExamples, p)
				mu.Unlock()
			case strings.Contains(lower, "license"):
				mu.Lock()
				sum.Licenses = append(sum.Licenses, p)
				mu.Unlock()
			case filepath.Base(lower) == "readme.md":
				if rs, err := parseReadmeSummary(full, cfg.MaxFileBytes); err == nil {
					mu.Lock()
					sum.Readmes = append(sum.Readmes, p)
					sum.ReadmeSummaries[p] = *rs
					mu.Unlock()
				} else {
					mu.Lock()
					sum.Readmes = append(sum.Readmes, p)
					mu.Unlock()
				}
			}

			// tech stats quick
			ext := filepath.Ext(lower)
			if ext == "" && strings.Contains(lower, "dockerfile") {
				ext = ".dockerfile"
			}
			mu.Lock()
			sum.TechStats[ext]++
			mu.Unlock()
		}()
	}
	wg.Wait()

	// Build pruned tree
	sum.Tree = buildTree(cfg.Root, cfg.TreeDepth, excludeGlobs)

	// Sort outputs
	sort.Slice(sum.GoModules, func(i, j int) bool { return sum.GoModules[i].Path < sum.GoModules[j].Path })
	sort.Strings(sum.MakeTargets)
	sort.Strings(sum.Dockerfiles)
	sort.Strings(sum.SQLMigrations)
	sort.Strings(sum.EnvExamples)
	sort.Strings(sum.Licenses)
	sort.Strings(sum.Readmes)

	return sum, nil
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseGoMod(path string) (*GoModule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	gm := &GoModule{Path: path}
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if strings.HasPrefix(ln, "module ") {
			gm.Module = strings.TrimSpace(strings.TrimPrefix(ln, "module"))
		}
		if strings.HasPrefix(ln, "require ") {
			rest := strings.TrimSpace(strings.TrimPrefix(ln, "require"))
			rest = strings.Trim(rest, "()")
			for _, r := range strings.Split(rest, "\n") {
				r = strings.TrimSpace(r)
				if r == "" || strings.HasPrefix(r, "//") {
					continue
				}
				parts := strings.Fields(r)
				if len(parts) >= 1 {
					gm.Requires = append(gm.Requires, parts[0])
				}
			}
		}
	}
	return gm, nil
}

// >>> Evitar conflito com built-in max (Go 1.21+)
func parseProto(path string, maxBytes int64) (*ProtoInfo, error) {
	head, err := files.ReadHead(path, maxBytes)
	if err != nil {
		return nil, err
	}
	pi := &ProtoInfo{File: path}
	for _, ln := range strings.Split(head, "\n") {
		ln = strings.TrimSpace(ln)
		if strings.HasPrefix(ln, "package ") {
			pi.Package = strings.TrimSuffix(strings.TrimSpace(strings.TrimPrefix(ln, "package")), ";")
		}
		if strings.HasPrefix(ln, "service ") {
			svc := strings.TrimSpace(strings.TrimPrefix(ln, "service"))
			svc = strings.TrimSpace(strings.SplitN(svc, "{", 2)[0])
			pi.Services = append(pi.Services, svc)
		}
		if strings.HasPrefix(ln, "rpc ") {
			rpc := strings.TrimSpace(strings.TrimPrefix(ln, "rpc"))
			rpc = strings.TrimSpace(strings.SplitN(rpc, "(", 2)[0])
			pi.RPCs = append(pi.RPCs, rpc)
		}
	}
	return pi, nil
}

func parseMakeTargets(path string, maxBytes int64) ([]string, error) {
	head, err := files.ReadHead(path, maxBytes)
	if err != nil {
		return nil, err
	}
	var targets []string
	for _, ln := range strings.Split(head, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		// simple heuristic: target: beginning of line and ends with colon
		if !strings.Contains(ln, "=") && strings.HasSuffix(ln, ":") && !strings.Contains(ln, " ") {
			t := strings.TrimSuffix(ln, ":")
			if t != "" {
				targets = append(targets, t)
			}
		}
	}
	return targets, nil
}

func parseDecision(path string, maxBytes int64) (*Decision, error) {
	head, err := files.ReadHead(path, maxBytes)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(head, "\n")
	title := ""
	summary := ""
	for _, ln := range lines {
		trim := strings.TrimSpace(ln)
		if strings.HasPrefix(trim, "# ") && title == "" {
			title = strings.TrimPrefix(trim, "# ")
			continue
		}
		if summary == "" && trim != "" && !strings.HasPrefix(trim, "#") {
			summary = trim
		}
		if title != "" && summary != "" {
			break
		}
	}
	return &Decision{File: path, Title: title, Summary: summary}, nil
}

func parseReadmeSummary(path string, max int64) (*ReadmeSummary, error) {
	head, err := files.ReadHead(path, max)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(head, "\n")

	var title string
	var firstPara string
	var objective string

	// 1) Título: pega primeiro "# " como H1
	for _, ln := range lines {
		trim := strings.TrimSpace(ln)
		if strings.HasPrefix(trim, "# ") {
			// corta possíveis marcadores de badges ao final
			title = strings.TrimSpace(strings.TrimPrefix(trim, "# "))
			break
		}
	}

	// 2) Primeiro parágrafo "livre" (ignora headings, listas e blocos vazios)
	//    Stop quando achar a primeira linha não vazia que não começa com '#' ou '-'
	for i := 0; i < len(lines); i++ {
		trim := strings.TrimSpace(lines[i])
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		if strings.HasPrefix(trim, "- ") || strings.HasPrefix(trim, "* ") || strings.HasPrefix(trim, ">") || strings.HasPrefix(trim, "`") {
			continue
		}
		firstPara = trim
		break
	}

	// 3) Objetivo/Objective: tenta localizar seção por heading e concatenar 1–3 linhas seguintes
	findSection := func(names ...string) string {
		for idx, ln := range lines {
			trim := strings.TrimSpace(ln)
			// matches "## Objetivo" / "## Objective" / "### Objetivo" etc.
			if strings.HasPrefix(trim, "##") {
				name := strings.TrimSpace(strings.TrimLeft(trim, "#"))
				name = strings.TrimSpace(name)
				for _, n := range names {
					if strings.EqualFold(name, n) {
						// coleta até 3 linhas "conteúdo" abaixo (pulando vazias e headers)
						var buf []string
						for j := idx + 1; j < len(lines) && len(buf) < 3; j++ {
							t := strings.TrimSpace(lines[j])
							if t == "" || strings.HasPrefix(t, "#") {
								continue
							}
							if strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ") {
								// junta bullets como frases curtas
								buf = append(buf, strings.TrimPrefix(strings.TrimPrefix(t, "- "), "* "))
								continue
							}
							// linha "normal"
							buf = append(buf, t)
						}
						return strings.Join(buf, " ")
					}
				}
			}
		}
		return ""
	}
	objective = findSection("Objetivo", "Objective", "Goals")

	// saneamento final (limita tamanho)
	limit := func(s string, n int) string {
		s = strings.ReplaceAll(s, "\r", " ")
		s = strings.ReplaceAll(s, "\n", " ")
		s = strings.TrimSpace(s)
		if len(s) > n {
			return s[:n] + "…"
		}
		return s
	}
	const maxLen = 400
	rs := &ReadmeSummary{
		File:      path,
		Title:     limit(title, 120),
		FirstPara: limit(firstPara, maxLen),
		Objective: limit(objective, maxLen),
	}
	return rs, nil
}

type treeNode struct {
	Name     string
	IsDir    bool
	Children []*treeNode
}

func buildTree(root string, depth int, exclude []string) []string {
	if depth <= 0 {
		depth = 3
	}
	// build nodes recursively
	var walk func(string, int) *treeNode
	walk = func(dir string, d int) *treeNode {
		node := &treeNode{Name: filepath.Base(dir), IsDir: true}
		if d == 0 {
			return node
		}
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".") && e.Name() != ".github" {
				continue
			}
			path := filepath.Join(dir, e.Name())
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			if e.IsDir() {
				if files.MatchAny(exclude, rel+"/") {
					continue
				}
				node.Children = append(node.Children, walk(path, d-1))
			} else {
				if files.MatchAny(exclude, rel) {
					continue
				}
				node.Children = append(node.Children, &treeNode{Name: e.Name(), IsDir: false})
			}
		}
		sort.Slice(node.Children, func(i, j int) bool { return node.Children[i].Name < node.Children[j].Name })
		return node
	}
	tree := walk(root, depth)
	var out []string
	var render func(n *treeNode, prefix string)
	render = func(n *treeNode, prefix string) {
		out = append(out, prefix+n.Name)
		for _, c := range n.Children {
			render(c, prefix+"  ")
		}
	}
	render(tree, "")
	return out
}

// MarshalJSON pretty JSON for Summary (useful for --out.json)
func (s *Summary) MarshalJSON() ([]byte, error) {
	type Alias Summary
	return json.MarshalIndent((*Alias)(s), "", "  ")
}
