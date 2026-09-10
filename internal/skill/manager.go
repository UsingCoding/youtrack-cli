package skill

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	youtrackcli "github.com/UsingCoding/youtrack-cli"
)

type Target struct {
	Name        string
	GlobalRoot  string
	ProjectRoot string
}

type Status struct {
	Target string `json:"target"`
	Path   string `json:"path"`
	Exists bool   `json:"installed"`
}

func Targets() ([]Target, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return []Target{
		{Name: "claude", GlobalRoot: filepath.Join(home, ".claude", "skills"), ProjectRoot: filepath.Join(".claude", "skills")},
		{Name: "codex", GlobalRoot: filepath.Join(home, ".codex", "skills"), ProjectRoot: filepath.Join(".codex", "skills")},
		{Name: "cursor", GlobalRoot: filepath.Join(home, ".cursor", "skills"), ProjectRoot: filepath.Join(".cursor", "skills")},
		{Name: "opencode", GlobalRoot: filepath.Join(home, ".config", "opencode", "skills"), ProjectRoot: filepath.Join(".opencode", "skills")},
		{Name: "agents", GlobalRoot: filepath.Join(home, ".agents", "skills"), ProjectRoot: filepath.Join(".agents", "skills")},
	}, nil
}

func List(project bool) ([]Status, error) {
	targets, err := Targets()
	if err != nil {
		return nil, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	out := make([]Status, 0, len(targets))
	for _, target := range targets {
		root := target.GlobalRoot
		if project {
			root = filepath.Join(cwd, target.ProjectRoot)
		}
		path := filepath.Join(root, "youtrack-cli")
		_, statErr := os.Stat(path)
		out = append(out, Status{Target: target.Name, Path: path, Exists: statErr == nil})
	}
	return out, nil
}

func Install(project bool, selected string) ([]string, error) {
	targets, err := Targets()
	if err != nil {
		return nil, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	var chosen []Target
	for _, target := range targets {
		if selected != "" {
			if target.Name == selected {
				chosen = append(chosen, target)
			}
			continue
		}
		root := target.GlobalRoot
		if project {
			root = filepath.Join(cwd, target.ProjectRoot)
		}
		if _, err := os.Stat(filepath.Dir(root)); err == nil {
			chosen = append(chosen, target)
		}
	}
	if selected != "" && len(chosen) == 0 {
		return nil, fmt.Errorf("unknown skill target %q", selected)
	}
	if len(chosen) == 0 {
		for _, target := range targets {
			if target.Name == "agents" {
				chosen = []Target{target}
				break
			}
		}
	}
	var installed []string
	for _, target := range chosen {
		root := target.GlobalRoot
		if project {
			root = filepath.Join(cwd, target.ProjectRoot)
		}
		dst := filepath.Join(root, "youtrack-cli")
		if err := copySkill(dst); err != nil {
			return nil, err
		}
		installed = append(installed, dst)
	}
	sort.Strings(installed)
	return installed, nil
}

func Remove(project bool, selected string) ([]string, error) {
	statuses, err := List(project)
	if err != nil {
		return nil, err
	}
	var removed []string
	for _, status := range statuses {
		if selected != "" && status.Target != selected {
			continue
		}
		if !status.Exists {
			continue
		}
		if err := os.RemoveAll(status.Path); err != nil {
			return nil, err
		}
		removed = append(removed, status.Path)
	}
	return removed, nil
}

func copySkill(dst string) error {
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil { // #nosec G301 -- Skill directories must be readable by coding agents.
		return err
	}
	return fs.WalkDir(youtrackcli.SkillsFS, "skills/youtrack-cli", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(path, "skills/youtrack-cli")
		rel = strings.TrimPrefix(rel, "/")
		if rel == "" {
			return nil
		}
		target := filepath.Join(dst, filepath.FromSlash(rel))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755) // #nosec G301 -- Skill directories must be readable by coding agents.
		}
		data, err := youtrackcli.SkillsFS.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644) // #nosec G306 -- Skill files are intentionally readable by coding agents.
	})
}
