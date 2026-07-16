// Command mira-mcp exposes the mira note-taking API as an MCP (Model Context
// Protocol) server over the stdio transport, so agents like Claude Code can
// search, read and create notes during a conversation.
//
// It never touches the database directly: every tool call goes through the
// same HTTP API the CLI (cmd/mira) uses, which is what guarantees that notes
// created by an agent get enriched asynchronously just like any other note.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"mira-tp4/internal/apiclient"
)

func main() {
	// slog goes to stderr only: stdout is reserved for JSON-RPC framing on
	// the stdio transport, and a single stray print there would corrupt the
	// protocol stream for the connected agent.
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	baseURL := os.Getenv("MIRA_API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	client := apiclient.New(baseURL)
	server := newServer(client, logger)

	logger.Info("mira-mcp starting", "mira_api_url", baseURL)
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		logger.Error("mira-mcp stopped", "error", err)
		os.Exit(1)
	}
}

// newServer builds the MCP server and registers all four tools against
// client. Split out from main so tests can construct a server wired to a
// fake HTTP API and drive it over an in-memory transport, without spawning
// the binary or needing a real mira API/Postgres.
func newServer(client *apiclient.Client, logger *slog.Logger) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mira-mcp",
		Title:   "Mira",
		Version: "1.0.0",
	}, &mcp.ServerOptions{
		Instructions: "Donne accès à la mémoire de notes de l'utilisateur (mira) : " +
			"recherche hybride, lecture et création de notes. Utilise search_notes " +
			"pour retrouver une note existante avant d'en créer une nouvelle sur le " +
			"même sujet, et add_note pour capturer un résumé ou une décision issue " +
			"de la conversation en cours.",
		Logger: logger,
	})

	registerTools(server, client, logger)
	return server
}
