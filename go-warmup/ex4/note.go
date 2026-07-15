package main

import "fmt"

// Note représente une note avec un titre, contenu et tags
type Note struct {
	ID      string
	Title   string
	Content string
	Tags    []string
}

// Preview retourne les 80 premiers caractères du contenu
func (n *Note) Preview() string {
	if len(n.Content) <= 80 {
		return n.Content
	}
	return n.Content[:80] + "..."
}

// String retourne une représentation textuelle de la note
func (n *Note) String() string {
	return fmt.Sprintf("ID: %s\nTitle: %s\nContent: %s\nTags: %v\n", n.ID, n.Title, n.Preview(), n.Tags)
}
