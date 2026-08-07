// Package skill discovers built-in (embedded) and user filesystem skills and
// builds a summary for the system prompt. Spec §6.6, §9.
package skill

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/OptLTD/swiflow/embed"
)

// Skill is a discovered skill.
type Skill struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"` // init|user
	Path        string `json:"path"`
	Body        string `json:"-"`
}

// Catalog discovers embedded built-in skills and optional user-dir overrides.
type Catalog struct {
	initFS  fs.FS
	initDir string // optional dev override (filesystem)
	userDir string
}

// NewCatalog creates a catalog. Built-in skills come from the embedded FS;
// userDir supplies overrides by slug. If devInitDir is non-empty, filesystem
// skills there override embedded builtins (for local development).
func NewCatalog(devInitDir, userDir string) *Catalog {
	return &Catalog{
		initFS:  embed.InitSkillsFS,
		initDir: devInitDir,
		userDir: userDir,
	}
}

// ForUserDir returns a shallow copy that discovers user skills from userDir.
func (c *Catalog) ForUserDir(userDir string) *Catalog {
	if c == nil {
		return NewCatalog("", userDir)
	}
	cp := *c
	cp.userDir = userDir
	return &cp
}

// Discover returns skills (embedded init first; dev init dir and user override by slug).
// Malformed entries are skipped.
func (c *Catalog) Discover(_ context.Context) []Skill {
	bySlug := map[string]Skill{}
	order := []string{}
	add := func(s Skill) {
		if _, ok := bySlug[s.Slug]; !ok {
			order = append(order, s.Slug)
		}
		bySlug[s.Slug] = s
	}
	for _, s := range discoverEmbed(c.initFS, "init-skills", "init") {
		add(s)
	}
	for _, s := range discoverDir(c.initDir, "init") {
		add(s)
	}
	for _, s := range discoverDir(c.userDir, "user") {
		add(s)
	}
	out := make([]Skill, 0, len(order))
	for _, slug := range order {
		out = append(out, bySlug[slug])
	}
	return out
}

// Summary returns a markdown list of skills (excluding disabled slugs).
func (c *Catalog) Summary(ctx context.Context, disabled map[string]bool) string {
	skills := c.Discover(ctx)
	if len(skills) == 0 {
		return ""
	}
	var b strings.Builder
	for _, s := range skills {
		if disabled[s.Slug] {
			continue
		}
		fmt.Fprintf(&b, "- %s: %s — %s\n", s.Slug, s.Name, s.Description)
	}
	return b.String()
}

func discoverEmbed(fsys fs.FS, root, source string) []Skill {
	entries, err := fs.ReadDir(fsys, root)
	if err != nil {
		return nil
	}
	var out []Skill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dirPath := path.Join(root, e.Name())
		s, ok := parseSkillFS(fsys, dirPath, source)
		if ok {
			out = append(out, s)
		}
	}
	return out
}

func discoverDir(dir, source string) []Skill {
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []Skill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		s, ok := parseSkill(filepath.Join(dir, e.Name()), source)
		if ok {
			out = append(out, s)
		}
	}
	return out
}

func parseSkillFS(fsys fs.FS, dirPath, source string) (Skill, bool) {
	for _, name := range []string{"SKILL.md", "skill.md"} {
		f := path.Join(dirPath, name)
		data, err := fs.ReadFile(fsys, f)
		if err != nil {
			continue
		}
		s, ok := parseFrontMatter(string(data))
		if !ok {
			return Skill{}, false
		}
		s.Source = source
		s.Path = "embed:" + dirPath
		return s, true
	}
	return Skill{}, false
}

func parseSkill(dirPath, source string) (Skill, bool) {
	for _, name := range []string{"SKILL.md", "skill.md"} {
		f := filepath.Join(dirPath, name)
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		s, ok := parseFrontMatter(string(data))
		if !ok {
			return Skill{}, false
		}
		s.Source = source
		s.Path = dirPath
		return s, true
	}
	return Skill{}, false
}

func parseFrontMatter(data string) (Skill, bool) {
	var s Skill
	lines := strings.Split(data, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return s, false
	}
	i := 1
	for i < len(lines) && strings.TrimSpace(lines[i]) != "---" {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "slug:") {
			s.Slug = strings.TrimSpace(strings.TrimPrefix(line, "slug:"))
		} else if strings.HasPrefix(line, "name:") {
			s.Name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		} else if strings.HasPrefix(line, "description:") {
			s.Description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		}
		i++
	}
	if i >= len(lines) {
		return s, false
	}
	s.Body = strings.TrimSpace(strings.Join(lines[i+1:], "\n"))
	if s.Slug == "" {
		return s, false
	}
	if s.Name == "" {
		s.Name = s.Slug
	}
	return s, true
}
