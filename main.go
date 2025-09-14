// Command llm-scan-tool scans a codebase and generates a Markdown summary
// suitable for providing context to LLMs. It also outputs a JSON summary.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/richardanchieta/llm-scan-tool/internal/collect"
	"github.com/richardanchieta/llm-scan-tool/internal/render"
)

func main() {
	var (
		root            string
		out             string
		maxFileBytes    int64
		threads         int
		includeGlobsStr string
		excludeGlobsStr string
		treeDepth       int
	)
	flag.StringVar(&root, "root", ".", "project root to scan")
	flag.StringVar(&out, "out", "LLM_SUMMARY.md", "output Markdown artifact path")
	flag.Int64Var(&maxFileBytes, "max-bytes-per-file", 64*1024, "max bytes to read from each file")
	flag.IntVar(&threads, "threads", runtime.NumCPU(), "max concurrent file reads")
	flag.StringVar(&includeGlobsStr, "include", "", "comma-separated glob patterns to force include (in addition to defaults)")
	flag.StringVar(&excludeGlobsStr, "exclude", "", "comma-separated glob patterns to exclude (in addition to defaults)")
	flag.IntVar(&treeDepth, "tree-depth", 3, "max depth for directory tree in the summary")
	flag.Parse()

	absRoot, err := filepath.Abs(root)
	if err != nil {
		log.Fatalf("resolve root: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	start := time.Now()
	cfg := collect.Config{
		Root:            absRoot,
		MaxFileBytes:    maxFileBytes,
		Threads:         threads,
		IncludeGlobsCSV: includeGlobsStr,
		ExcludeGlobsCSV: excludeGlobsStr,
		TreeDepth:       treeDepth,
	}
	sum, err := collect.Scan(ctx, cfg)
	if err != nil {
		log.Fatalf("scan failed: %v", err)
	}

	md, j, err := render.BuildArtifacts(sum)
	if err != nil {
		log.Fatalf("render failed: %v", err)
	}

	if err := os.WriteFile(out, []byte(md), 0o644); err != nil {
		log.Fatalf("write markdown: %v", err)
	}
	jsonPath := out + ".json"
	if err := os.WriteFile(jsonPath, j, 0o644); err != nil {
		log.Fatalf("write json: %v", err)
	}

	fmt.Printf("Generated %s and %s in %s\n", out, jsonPath, time.Since(start))
}
