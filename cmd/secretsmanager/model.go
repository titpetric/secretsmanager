package main

import (
	"errors"
	"os"

	"encoding/json"

	"github.com/blaskovicz/go-cryptkeeper"
	"github.com/m4rw3r/uuid"
)

type Storage struct {
	filename string
	loaded   bool

	Secrets []*Secret `json:"secrets,omitempty"`
}

func NewStorage(filename string) (*Storage, error) {
	storage := &Storage{
		filename: filename,
	}
	return storage, nil
}

func (s *Storage) Load() error {
	key := os.Getenv("SECRETSMANAGER_KEY")
	if len(key) != 32 {
		return errors.New("Invalid or missing SECRETSMANAGER_KEY")
	}

	if err := cryptkeeper.SetCryptKey([]byte(key)); err != nil {
		return err
	}

	f, err := os.Open(s.filename)
	if err != nil {
		return err
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	if err := decoder.Decode(s); err != nil {
		return err
	}
	s.loaded = true
	return nil
}

func (s *Storage) Save() error {
	if !s.loaded {
		return nil
	}
	tmpFilename := s.filename + ".tmp"
	f, err := os.Create(tmpFilename)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(s); err != nil {
		return err
	}

	return os.Rename(tmpFilename, s.filename)
}

type Secret struct {
	ID    uuid.UUID
	Name  string
	Value cryptkeeper.CryptString
}

func NewSecret(name, value string) (*Secret, error) {
	id, err := uuid.V4()
	if err != nil {
		return nil, err
	}

	return &Secret{
		ID:    id,
		Name:  name,
		Value: cryptkeeper.CryptString{String: value},
	}, nil
}
