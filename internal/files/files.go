package files

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DefaultIgnore returns common directories/files to skip for LLM-oriented scans.
func DefaultIgnore() []string {
	return []string{
		".git/", ".git/**",
		".hg/", ".svn/",
		"node_modules/**", "dist/**", "build/**", "out/**", "bin/**", "obj/**",
		".venv/**", "venv/**", "__pycache__/**",
		".idea/**", ".vscode/**", ".DS_Store",
		".terraform/**", "vendor/**",
		"*.png", "*.jpg", "*.jpeg", "*.gif", "*.webp", "*.ico",
		"*.pdf", "*.ppt", "*.pptx", "*.doc", "*.docx", "*.xls", "*.xlsx",
		"*.zip", "*.tar", "*.gz", "*.tgz", "*.7z",
		"*.mp4", "*.mp3", "*.mov", "*.avi",
	}
}

func ReadHead(path string, max int64) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var b strings.Builder
	r := bufio.NewReader(f)
	var n int64
	for {
		if n >= max {
			break
		}
		chunk := int64(4096)
		if max-n < chunk {
			chunk = max - n
		}
		buf := make([]byte, chunk)
		read, err := r.Read(buf)
		if read > 0 {
			b.Write(buf[:read])
			n += int64(read)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return b.String(), err
		}
	}
	return b.String(), nil
}

func MatchAny(globs []string, path string) bool {
	for _, g := range globs {
		ok, _ := filepath.Match(g, path)
		if ok {
			return true
		}
		// also try on base name
		if ok, _ := filepath.Match(g, filepath.Base(path)); ok {
			return true
		}
	}
	return false
}
