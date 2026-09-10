package credentials

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type entry struct {
	Token string `toml:"token"`
}
type fileData struct {
	Profiles map[string]entry `toml:"profiles"`
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
	return filepath.Join(root, "youtrack", "credentials.toml"), nil
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

func (s *Store) load() (fileData, error) {
	data := fileData{Profiles: map[string]entry{}}
	_, err := os.Stat(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return data, nil
	}
	if err != nil {
		return fileData{}, err
	}
	if _, err := toml.DecodeFile(s.Path, &data); err != nil {
		return fileData{}, err
	}
	if data.Profiles == nil {
		data.Profiles = map[string]entry{}
	}
	return data, nil
}

func (s *Store) save(data fileData) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.Path), ".credentials-*.toml")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer func() { _ = os.Remove(name) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := toml.NewEncoder(tmp).Encode(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, s.Path); err != nil {
		return err
	}
	return os.Chmod(s.Path, 0o600)
}

func (s *Store) Get(profile string) (string, error) {
	data, err := s.load()
	if err != nil {
		return "", err
	}
	return data.Profiles[profile].Token, nil
}

func (s *Store) Set(profile, token string) error {
	data, err := s.load()
	if err != nil {
		return err
	}
	data.Profiles[profile] = entry{Token: token}
	return s.save(data)
}

func (s *Store) Delete(profile string) error {
	data, err := s.load()
	if err != nil {
		return err
	}
	delete(data.Profiles, profile)
	return s.save(data)
}
