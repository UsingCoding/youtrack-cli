package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Profile struct {
	URL string `toml:"url"`
}

type Config struct {
	Current  string             `toml:"current"`
	Profiles map[string]Profile `toml:"profiles"`
}

type Store struct{ Path string }

func DefaultPath() (string, error) {
	root := os.Getenv("XDG_CONFIG_HOME")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		root = filepath.Join(home, ".config")
	}
	return filepath.Join(root, "youtrack", "config.toml"), nil
}

func NewStore(path string) (*Store, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, err
		}
	}
	return &Store{Path: path}, nil
}

func (s *Store) Load() (Config, error) {
	cfg := Config{Profiles: map[string]Profile{}}
	_, err := os.Stat(s.Path) // #nosec G703 -- Store.Path is intentionally caller-configurable.
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return Config{}, err
	}
	if _, err := toml.DecodeFile(s.Path, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	return cfg, nil
}

func (s *Store) Save(cfg Config) error {
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.Path), ".config-*.toml")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer func() { _ = os.Remove(name) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	enc := toml.NewEncoder(tmp)
	if err := enc.Encode(cfg); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, s.Path)
}

func (s *Store) Set(key, value string) error {
	cfg, err := s.Load()
	if err != nil {
		return err
	}
	if key == "current" {
		if _, ok := cfg.Profiles[value]; !ok {
			return fmt.Errorf("profile %q does not exist", value)
		}
		cfg.Current = value
		return s.Save(cfg)
	}
	parts := strings.Split(key, ".")
	if len(parts) == 3 && parts[0] == "profile" && parts[2] == "url" {
		p := cfg.Profiles[parts[1]]
		p.URL = value
		cfg.Profiles[parts[1]] = p
		if cfg.Current == "" {
			cfg.Current = parts[1]
		}
		return s.Save(cfg)
	}
	return fmt.Errorf("unsupported config key %q", key)
}

func (s *Store) Get(key string) (string, error) {
	cfg, err := s.Load()
	if err != nil {
		return "", err
	}
	if key == "current" {
		return cfg.Current, nil
	}
	parts := strings.Split(key, ".")
	if len(parts) == 3 && parts[0] == "profile" && parts[2] == "url" {
		p, ok := cfg.Profiles[parts[1]]
		if !ok {
			return "", fmt.Errorf("profile %q does not exist", parts[1])
		}
		return p.URL, nil
	}
	return "", fmt.Errorf("unsupported config key %q", key)
}
