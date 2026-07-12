package skill

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var slugRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// ValidSlug reports whether slug is safe for use as a directory name.
func ValidSlug(slug string) error {
	if slug == "" {
		return fmt.Errorf("slug is required")
	}
	if len(slug) > 64 {
		return fmt.Errorf("slug too long")
	}
	if !slugRE.MatchString(slug) {
		return fmt.Errorf("slug must be lowercase alphanumeric with hyphens")
	}
	return nil
}

// FormatSkillMD renders a skill as a SKILL.md file.
func FormatSkillMD(s Skill) string {
	name := s.Name
	if name == "" {
		name = s.Slug
	}
	desc := s.Description
	return fmt.Sprintf("---\nslug: %s\nname: %s\ndescription: %s\n---\n\n%s", s.Slug, name, desc, s.Body)
}

// Get returns a discovered skill by slug.
func (c *Catalog) Get(slug string) (Skill, bool) {
	for _, s := range c.Discover(context.Background()) {
		if s.Slug == slug {
			return s, true
		}
	}
	return Skill{}, false
}

// ReadSkillMD returns the full SKILL.md content for a skill. User skills are read
// from disk; built-in skills are reconstructed from parsed metadata.
func (c *Catalog) ReadSkillMD(slug string) (string, error) {
	if err := ValidSlug(slug); err != nil {
		return "", err
	}
	userFile := c.userSkillFile(slug)
	if data, err := os.ReadFile(userFile); err == nil {
		return string(data), nil
	}
	s, ok := c.Get(slug)
	if !ok {
		return "", fmt.Errorf("skill not found: %s", slug)
	}
	return FormatSkillMD(s), nil
}

// CreateSkill writes a new user skill. Fails if the user copy already exists.
func (c *Catalog) CreateSkill(slug, content string) error {
	if c.userDir == "" {
		return fmt.Errorf("user skills directory is not configured")
	}
	if err := ValidSlug(slug); err != nil {
		return err
	}
	parsed, ok := parseFrontMatter(content)
	if !ok {
		return fmt.Errorf("invalid SKILL.md: missing front matter (slug, name, description)")
	}
	if parsed.Slug != slug {
		return fmt.Errorf("front matter slug %q does not match %q", parsed.Slug, slug)
	}
	userFile := c.userSkillFile(slug)
	if _, err := os.Stat(userFile); err == nil {
		return fmt.Errorf("skill %s already exists; use patch to update", slug)
	}
	return writeSkillFile(userFile, content)
}

// PatchSkill applies a targeted edit and writes the user skill copy (overriding built-ins).
func (c *Catalog) PatchSkill(slug, oldString, newString string, replaceAll bool) error {
	if c.userDir == "" {
		return fmt.Errorf("user skills directory is not configured")
	}
	if err := ValidSlug(slug); err != nil {
		return err
	}
	if oldString == "" {
		return fmt.Errorf("old_string is required")
	}
	current, err := c.ReadSkillMD(slug)
	if err != nil {
		return err
	}
	if !strings.Contains(current, oldString) {
		return fmt.Errorf("old_string not found in skill %s", slug)
	}
	var updated string
	if replaceAll {
		updated = strings.ReplaceAll(current, oldString, newString)
	} else {
		n := strings.Count(current, oldString)
		if n > 1 {
			return fmt.Errorf("old_string matches %d times; set replace_all or use a more specific snippet", n)
		}
		updated = strings.Replace(current, oldString, newString, 1)
	}
	if _, ok := parseFrontMatter(updated); !ok {
		return fmt.Errorf("patch would produce invalid SKILL.md")
	}
	return writeSkillFile(c.userSkillFile(slug), updated)
}

func (c *Catalog) userSkillFile(slug string) string {
	return filepath.Join(c.userDir, slug, "SKILL.md")
}

func writeSkillFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create skill dir: %w", err)
	}
	normalized := strings.TrimRight(content, "\n") + "\n"
	if err := os.WriteFile(path, []byte(normalized), 0o644); err != nil {
		return fmt.Errorf("write skill: %w", err)
	}
	return nil
}
