// Package files contém utilitários de leitura segura/eficiente de arquivos.
package files

import (
	"bufio"
	"errors"
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

// ReadHead lê até maxBytes do início do arquivo (head) e retorna como string.
// Útil para limitação de contexto em arquivos grandes.
func ReadHead(path string, maxBytes int64) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	// garantir errcheck no Close
	defer func() { _ = f.Close() }()

	if maxBytes <= 0 {
		return "", nil
	}

	r := bufio.NewReader(f)
	var (
		varBuf  = make([]byte, 32*1024)
		total   int64
		builder strings.Builder
	)

	// evita QF1006: já checa no header do for
	for total < maxBytes {
		toRead := int64(len(varBuf))
		if rem := maxBytes - total; rem < toRead {
			toRead = rem
		}
		n, readErr := io.ReadFull(r, varBuf[:toRead])
		if n > 0 {
			builder.Write(varBuf[:n])
			total += int64(n)
		}
		if errors.Is(readErr, io.EOF) || errors.Is(readErr, io.ErrUnexpectedEOF) {
			break
		}
		if readErr != nil {
			return builder.String(), readErr
		}
	}
	return builder.String(), nil
}

// MatchAny verifica se path casa com qualquer um dos globs fornecidos.
func MatchAny(globs []string, path string) bool {
	if len(globs) == 0 {
		return false
	}
	for _, g := range globs {
		if ok, _ := filepath.Match(g, path); ok {
			return true
		}
	}
	return false
}
