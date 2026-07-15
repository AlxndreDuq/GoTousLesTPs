package notes

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
)

// JSONLStore est une implémentation de NoteStore qui persiste les notes
// en JSON Lines, une note par ligne, dans un fichier sur disque.
type JSONLStore struct {
	Path string
}

// NewJSONLStore crée un store pointant vers ~/.mira/notes.jsonl.
func NewJSONLStore() (*JSONLStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return &JSONLStore{Path: filepath.Join(home, ".mira", "notes.jsonl")}, nil
}

// Add ajoute une note en fin de fichier, créant le dossier et le fichier
// si nécessaire.
func (s *JSONLStore) Add(note Note) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return err
	}

	f, err := os.OpenFile(s.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	line, err := json.Marshal(note)
	if err != nil {
		return err
	}

	_, err = f.Write(append(line, '\n'))
	return err
}

// All retourne toutes les notes stockées, dans leur ordre d'écriture.
func (s *JSONLStore) All() ([]Note, error) {
	f, err := os.Open(s.Path)
	if os.IsNotExist(err) {
		return []Note{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var result []Note
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var n Note
		if err := json.Unmarshal(line, &n); err != nil {
			return nil, err
		}
		result = append(result, n)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// List retourne les `limit` notes les plus récentes, de la plus ancienne
// à la plus récente.
func (s *JSONLStore) List(limit int) ([]Note, error) {
	all, err := s.All()
	if err != nil {
		return nil, err
	}

	if limit <= 0 || limit >= len(all) {
		return all, nil
	}

	return all[len(all)-limit:], nil
}
