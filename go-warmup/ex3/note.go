package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Note représente une note avec un titre, contenu et tags
type Note struct {
	Title   string
	Content string
	Tags    []string
}

// NewNote crée une nouvelle note avec titre et contenu
func NewNote(title, content string) *Note {
	return &Note{
		Title:   title,
		Content: content,
		Tags:    []string{},
	}
}

// Preview retourne les 80 premiers caractères du contenu
func (n *Note) Preview() string {
	if len(n.Content) <= 80 {
		return n.Content
	}
	return n.Content[:80] + "..."
}

// AddTag ajoute un tag à la note si ce n'est pas déjà présent (pas de doublons)
func (n *Note) AddTag(tag string) {
	for _, t := range n.Tags {
		if t == tag {
			return // Tag déjà présent
		}
	}
	n.Tags = append(n.Tags, tag)
}

// LoadFromFile charge des notes depuis un fichier JSON
func LoadFromFile(path string) ([]*Note, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var notes []*Note
	err = json.NewDecoder(file).Decode(&notes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return notes, nil
}

// HasTag retourne true si la note contient le tag donné
func (n *Note) HasTag(tag string) bool {
	for _, t := range n.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// String retourne une représentation textuelle de la note
func (n *Note) String() string {
	return fmt.Sprintf("Title: %s\nPreview: %s\nTags: %s\n", n.Title, n.Preview(), strings.Join(n.Tags, ", "))
}
