package settings

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
)

var (
	ErrInvalid  = errors.New("invalid setting")
	ErrNotFound = errors.New("setting not found")
	keyPattern  = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
)

type Store struct{ directory string }

func Open(directory string) (*Store, error) {
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	return &Store{directory: directory}, nil
}

func (s *Store) Set(key string, value int) error {
	if !keyPattern.MatchString(key) || value < 0 {
		return ErrInvalid
	}
	data, err := json.Marshal(struct {
		Value int `json:"value"`
	}{value})
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.directory, key+".json"), data, 0600)
}

func (s *Store) Get(key string) (int, error) {
	if !keyPattern.MatchString(key) {
		return 0, ErrInvalid
	}
	data, err := os.ReadFile(filepath.Join(s.directory, key+".json"))
	if errors.Is(err, os.ErrNotExist) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	var record struct {
		Value int `json:"value"`
	}
	if err := json.Unmarshal(data, &record); err != nil {
		return 0, err
	}
	return record.Value, nil
}
