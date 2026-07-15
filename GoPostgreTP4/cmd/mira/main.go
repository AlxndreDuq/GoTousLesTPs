package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"mira-tp4/internal/apiclient"
	"mira-tp4/internal/core"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	baseURL := os.Getenv("MIRA_API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	client := apiclient.New(baseURL)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var cmdErr error
	switch os.Args[1] {
	case "add":
		cmdErr = runAdd(ctx, client, os.Args[2:])
	case "list":
		cmdErr = runList(ctx, client)
	case "search":
		cmdErr = runSearch(ctx, client, os.Args[2:])
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

func runAdd(ctx context.Context, client *apiclient.Client, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf(`usage: mira add "titre" "contenu"`)
	}

	note, err := client.CreateNote(ctx, args[0], args[1])
	if err != nil {
		return err
	}

	fmt.Printf("note ajoutée (%s) — enrichissement %s\n", note.ID, note.EnrichmentStatus)
	return nil
}

func runList(ctx context.Context, client *apiclient.Client) error {
	notes, err := client.ListRecent(ctx, 10)
	if err != nil {
		return err
	}

	if len(notes) == 0 {
		fmt.Println("aucune note")
		return nil
	}

	for _, n := range notes {
		printNote(n)
	}
	return nil
}

func runSearch(ctx context.Context, client *apiclient.Client, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mira search <query>")
	}
	query := strings.Join(args, " ")

	notes, err := client.SearchNotes(ctx, query)
	if err != nil {
		return err
	}

	if len(notes) == 0 {
		fmt.Println("aucun résultat")
		return nil
	}

	for _, n := range notes {
		printNote(n)
	}
	return nil
}

func printNote(n core.Note) {
	fmt.Printf("[%s] %s\n    %s\n", n.CreatedAt.Format("2006-01-02 15:04"), n.Title, n.Content)
	if len(n.Tags) > 0 {
		fmt.Printf("    tags: %s\n", strings.Join(n.Tags, ", "))
	}
}
