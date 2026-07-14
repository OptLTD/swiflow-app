package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/OptLTD/swiflow/internal/util"
)

// Draft is a skill proposal awaiting human confirmation.
type Draft struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Content   string `json:"content"`
	Note      string `json:"note,omitempty"`
	CreatedAt string `json:"created_at"`
}

func (c *Catalog) draftsDir() string {
	if c.userDir == "" {
		return ""
	}
	return filepath.Join(c.userDir, ".drafts")
}

// SaveDraft stores a draft SKILL.md for human review.
func (c *Catalog) SaveDraft(slug, content, note string) (Draft, error) {
	if c.userDir == "" {
		return Draft{}, fmt.Errorf("user skills directory is not configured")
	}
	if err := ValidSlug(slug); err != nil {
		return Draft{}, err
	}
	parsed, ok := parseFrontMatter(content)
	if !ok {
		return Draft{}, fmt.Errorf("invalid SKILL.md: missing front matter")
	}
	if parsed.Slug != "" && parsed.Slug != slug {
		return Draft{}, fmt.Errorf("front matter slug %q does not match %q", parsed.Slug, slug)
	}
	id := util.NewID()
	dir := filepath.Join(c.draftsDir(), id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Draft{}, err
	}
	path := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(path, []byte(strings.TrimRight(content, "\n")+"\n"), 0o644); err != nil {
		return Draft{}, err
	}
	meta := fmt.Sprintf("slug=%s\nnote=%s\n", slug, strings.ReplaceAll(note, "\n", " "))
	_ = os.WriteFile(filepath.Join(dir, "meta.txt"), []byte(meta), 0o644)
	return Draft{
		ID:        id,
		Slug:      slug,
		Content:   content,
		Note:      note,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// ListDrafts returns pending skill drafts.
func (c *Catalog) ListDrafts() ([]Draft, error) {
	dir := c.draftsDir()
	if dir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Draft
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		d, err := c.readDraft(e.Name())
		if err != nil {
			continue
		}
		out = append(out, d)
	}
	return out, nil
}

func (c *Catalog) readDraft(id string) (Draft, error) {
	dir := filepath.Join(c.draftsDir(), id)
	content, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		return Draft{}, err
	}
	slug := ""
	note := ""
	if meta, err := os.ReadFile(filepath.Join(dir, "meta.txt")); err == nil {
		for _, line := range strings.Split(string(meta), "\n") {
			if strings.HasPrefix(line, "slug=") {
				slug = strings.TrimPrefix(line, "slug=")
			}
			if strings.HasPrefix(line, "note=") {
				note = strings.TrimPrefix(line, "note=")
			}
		}
	}
	info, _ := os.Stat(filepath.Join(dir, "SKILL.md"))
	created := ""
	if info != nil {
		created = info.ModTime().UTC().Format(time.RFC3339)
	}
	return Draft{ID: id, Slug: slug, Content: string(content), Note: note, CreatedAt: created}, nil
}

// AcceptDraft promotes a draft into a user skill and removes the draft.
func (c *Catalog) AcceptDraft(id string) error {
	d, err := c.readDraft(id)
	if err != nil {
		return err
	}
	if d.Slug == "" {
		return fmt.Errorf("draft missing slug")
	}
	// Overwrite allowed on accept if user confirms.
	userFile := c.userSkillFile(d.Slug)
	if err := writeSkillFile(userFile, d.Content); err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(c.draftsDir(), id))
}

// DeleteDraft removes a pending draft.
func (c *Catalog) DeleteDraft(id string) error {
	dir := filepath.Join(c.draftsDir(), id)
	return os.RemoveAll(dir)
}
