package files

import (
	"os"
	"path/filepath"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

// GitIgnoreMatcher verifica se um arquivo deve ser ignorado com base em .gitignore.
type GitIgnoreMatcher struct {
	ignores []*ignore.GitIgnore
}

// NewGitIgnoreMatcher lê todos os .gitignore encontrados até a raiz.
func NewGitIgnoreMatcher(root string) *GitIgnoreMatcher {
	var patterns []*ignore.GitIgnore
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if strings.HasSuffix(info.Name(), ".gitignore") {
			if g, e := ignore.CompileIgnoreFile(path); e == nil {
				patterns = append(patterns, g)
			}
		}
		return nil
	})
	return &GitIgnoreMatcher{ignores: patterns}
}

// Match retorna true se path deve ser ignorado.
func (g *GitIgnoreMatcher) Match(path string) bool {
	for _, ig := range g.ignores {
		if ig.MatchesPath(path) {
			return true
		}
	}
	return false
}
