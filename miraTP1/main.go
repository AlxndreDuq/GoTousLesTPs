package main

import (
	"fmt"
	"os"
	"strings"

	"mira/internal/notes"
	"mira/internal/search"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	store, err := notes.NewJSONLStore()
	if err != nil {
		fmt.Fprintln(os.Stderr, "mira:", err)
		os.Exit(1)
	}

	var cmdErr error
	switch os.Args[1] {
	case "add":
		cmdErr = runAdd(store, os.Args[2:])
	case "list":
		cmdErr = runList(store)
	case "search":
		cmdErr = runSearch(store, os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}

	if cmdErr != nil {
		fmt.Fprintln(os.Stderr, "mira:", cmdErr)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage:
  mira add "titre" "contenu"
  mira list
  mira search <query>`)
}

func runAdd(store notes.NoteStore, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf(`usage: mira add "titre" "contenu"`)
	}

	note := notes.NewNote(args[0], args[1])
	if err := store.Add(note); err != nil {
		return err
	}

	fmt.Printf("note ajoutée (%d)\n", note.ID)
	return nil
}

func runList(store notes.NoteStore) error {
	list, err := store.List(10)
	if err != nil {
		return err
	}

	if len(list) == 0 {
		fmt.Println("aucune note")
		return nil
	}

	for _, n := range list {
		printNote(n)
	}
	return nil
}

func runSearch(store notes.NoteStore, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mira search <query>")
	}
	query := strings.Join(args, " ")

	all, err := store.All()
	if err != nil {
		return err
	}

	results := search.Search(all, query)
	if len(results) == 0 {
		fmt.Println("aucun résultat")
		return nil
	}

	for _, n := range results {
		printNote(n)
	}
	return nil
}

func printNote(n notes.Note) {
	fmt.Printf("[%s] %s\n    %s\n", n.CreatedAt.Format("2006-01-02 15:04"), n.Title, n.Content)
}
