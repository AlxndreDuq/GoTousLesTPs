package main

import (
	"fmt"
	"log"
)

func main() {
	// Test 1 : Créer des notes manuellement
	note1 := NewNote("Go Basics", "This is a comprehensive guide to Go programming. Learn about variables, types, and functions.")
	note1.AddTag("go")
	note1.AddTag("programming")
	note1.AddTag("tutorial")

	note2 := NewNote("Web Development", "Build web applications using Go and various frameworks available in the ecosystem.")
	note2.AddTag("go")
	note2.AddTag("web")
	note2.AddTag("backend")

	note3 := NewNote("Python Tips", "Best practices for Python programming and data science applications.")
	note3.AddTag("python")
	note3.AddTag("programming")

	fmt.Println("=== Manually created notes ===")
	fmt.Println("All notes:")
	for i, note := range []*Note{note1, note2, note3} {
		fmt.Printf("\nNote %d:\n%s\n", i+1, note)
	}

	// Test 2 : Filtrer les notes contenant le tag "go"
	fmt.Println("\n=== Notes containing tag 'go' ===")
	for _, note := range []*Note{note1, note2, note3} {
		if note.HasTag("go") {
			fmt.Printf("Title: %s\n", note.Title)
			fmt.Printf("Preview: %s\n", note.Preview())
			fmt.Printf("Tags: %v\n\n", note.Tags)
		}
	}

	// Test 3 : Tenter de charger depuis un fichier (le fichier notes.json doit exister)
	fmt.Println("=== Loading from file ===")
	notes, err := LoadFromFile("notes.json")
	if err != nil {
		log.Printf("Could not load from file: %v (this is normal if notes.json doesn't exist)\n", err)
	} else {
		fmt.Printf("Loaded %d notes from file\n", len(notes))
		for _, note := range notes {
			if note.HasTag("go") {
				fmt.Printf("Title: %s\n", note.Title)
			}
		}
	}
}
