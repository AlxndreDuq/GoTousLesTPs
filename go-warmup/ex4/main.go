package main

import (
	"errors"
	"fmt"
)

func main() {
	store := NewMemoryStore()

	// Test 1 : Save une note valide
	fmt.Println("=== Test 1: Save valid note ===")
	note1 := &Note{
		ID:      "1",
		Title:   "Go Basics",
		Content: "Introduction to Go programming language",
		Tags:    []string{"go", "tutorial"},
	}
	err := store.Save(note1)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Note saved successfully")
	}

	// Test 2 : Save une note avec titre vide (doit retourner ErrValidation)
	fmt.Println("\n=== Test 2: Save note with empty title ===")
	note2 := &Note{
		ID:      "2",
		Title:   "   ", // titre vide/whitespace
		Content: "This should fail",
		Tags:    []string{},
	}
	err = store.Save(note2)
	if err != nil {
		if errors.Is(err, ErrValidation) {
			fmt.Printf("Got expected error: %v\n", err)
		} else {
			fmt.Printf("Unexpected error: %v\n", err)
		}
	}

	// Test 3 : Save un doublon (doit retourner ErrDuplicate)
	fmt.Println("\n=== Test 3: Save duplicate note ===")
	note1Dup := &Note{
		ID:      "1", // Même ID que note1
		Title:   "Different Title",
		Content: "Different content",
		Tags:    []string{},
	}
	err = store.Save(note1Dup)
	if err != nil {
		if errors.Is(err, ErrDuplicate) {
			fmt.Printf("Got expected error: %v\n", err)
		} else {
			fmt.Printf("Unexpected error: %v\n", err)
		}
	}

	// Test 4 : Get une note existante
	fmt.Println("\n=== Test 4: Get existing note ===")
	retrieved, err := store.Get("1")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Retrieved note: %s\n", retrieved.Title)
	}

	// Test 5 : Get une note inexistante (doit retourner ErrNotFound)
	fmt.Println("\n=== Test 5: Get non-existing note ===")
	_, err = store.Get("999")
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			fmt.Printf("Got expected error: %v\n", err)
		} else {
			fmt.Printf("Unexpected error: %v\n", err)
		}
	}

	// Test 6 : Save d'autres notes et All()
	fmt.Println("\n=== Test 6: All notes ===")
	note3 := &Note{
		ID:      "3",
		Title:   "Go Advanced",
		Content: "Advanced Go concepts and patterns",
		Tags:    []string{"go", "advanced"},
	}
	store.Save(note3)

	allNotes := store.All()
	fmt.Printf("Total notes in store: %d\n", len(allNotes))
	for _, note := range allNotes {
		fmt.Printf("- %s (ID: %s)\n", note.Title, note.ID)
	}
}
