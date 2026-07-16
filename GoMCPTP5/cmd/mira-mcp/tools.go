package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"mira-tp4/internal/apiclient"
	"mira-tp4/internal/core"
)

// apiCallTimeout bounds every underlying HTTP call to the mira API,
// independently of whatever deadline (if any) the connected agent set on
// the tool call itself.
const apiCallTimeout = 10 * time.Second

const (
	defaultSearchLimit = 10
	maxSearchLimit     = 50 // mira's hybrid search endpoint caps results at 50 server-side.

	defaultRecentLimit = 10
	maxRecentLimit     = 100 // mirrors the API's own list pagination cap.
)

// notesListOutput is the structured output shared by the two tools that
// return several notes (search_notes, list_recent_notes).
type notesListOutput struct {
	Notes []core.Note `json:"notes"`
	Count int         `json:"count"`
}

func registerTools(server *mcp.Server, client *apiclient.Client, logger *slog.Logger) {
	registerSearchNotes(server, client, logger)
	registerGetNote(server, client, logger)
	registerAddNote(server, client, logger)
	registerListRecentNotes(server, client, logger)
}

type searchNotesArgs struct {
	Query string `json:"query" jsonschema:"Termes ou phrase à rechercher. Combine une recherche plein texte et une similarité vectorielle sur les notes déjà enrichies : fonctionne aussi bien avec des mots-clés exacts qu'avec une description approximative du contenu recherché."`
	Limit int    `json:"limit,omitempty" jsonschema:"Nombre maximum de notes à retourner. Par défaut 10, plafonné à 50."`
}

func registerSearchNotes(server *mcp.Server, client *apiclient.Client, logger *slog.Logger) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "search_notes",
		Description: "Recherche des notes existantes dans la mémoire de l'utilisateur (mira) par mots-clés " +
			"ou par similarité de sens. À utiliser en premier pour retrouver une note avant d'en recréer " +
			"une nouvelle sur le même sujet, ou pour répondre à une question sur ce que l'utilisateur a " +
			"déjà noté.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args searchNotesArgs) (*mcp.CallToolResult, notesListOutput, error) {
		query := strings.TrimSpace(args.Query)
		if query == "" {
			return nil, notesListOutput{}, errors.New("query is required and cannot be empty")
		}
		limit, err := normalizeLimit(args.Limit, defaultSearchLimit, maxSearchLimit)
		if err != nil {
			return nil, notesListOutput{}, err
		}

		callCtx, cancel := context.WithTimeout(ctx, apiCallTimeout)
		defer cancel()

		notes, err := client.SearchNotes(callCtx, query)
		if err != nil {
			logger.Error("search_notes: api call failed", "query", query, "error", err)
			return nil, notesListOutput{}, describeAPIError(err)
		}
		if len(notes) > limit {
			notes = notes[:limit]
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: formatNotesList(notes)}},
		}, notesListOutput{Notes: notes, Count: len(notes)}, nil
	})
}

type getNoteArgs struct {
	ID string `json:"id" jsonschema:"Identifiant (UUID) de la note à récupérer, tel que retourné par search_notes, add_note ou list_recent_notes."`
}

func registerGetNote(server *mcp.Server, client *apiclient.Client, logger *slog.Logger) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_note",
		Description: "Récupère une note complète (contenu intégral, tags, résumé généré, score, statut " +
			"d'enrichissement) à partir de son identifiant. À utiliser après search_notes ou " +
			"list_recent_notes lorsque le contenu complet d'une note précise est nécessaire.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args getNoteArgs) (*mcp.CallToolResult, core.Note, error) {
		id := strings.TrimSpace(args.ID)
		if id == "" {
			return nil, core.Note{}, errors.New("id is required and cannot be empty")
		}

		callCtx, cancel := context.WithTimeout(ctx, apiCallTimeout)
		defer cancel()

		note, err := client.GetNote(callCtx, id)
		if err != nil {
			logger.Error("get_note: api call failed", "id", id, "error", err)
			return nil, core.Note{}, describeAPIError(err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: formatNote(note)}},
		}, note, nil
	})
}

type addNoteArgs struct {
	Title   string   `json:"title" jsonschema:"Titre court de la note."`
	Content string   `json:"content" jsonschema:"Contenu de la note. Peut être un extrait de conversation, un résumé ou toute information à mémoriser."`
	Tags    []string `json:"tags,omitempty" jsonschema:"Tags optionnels à associer à la note en plus de ceux que l'enrichissement automatique générera."`
}

func registerAddNote(server *mcp.Server, client *apiclient.Client, logger *slog.Logger) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "add_note",
		Description: "Crée une nouvelle note dans la mémoire de l'utilisateur (mira). La note est enregistrée " +
			"immédiatement puis enrichie de façon asynchrone côté serveur (tags, résumé, score) ; juste après " +
			"la création, enrichment_status vaut généralement \"pending\" et passe à \"done\" quelques instants " +
			"plus tard (relire la note via get_note pour vérifier). À utiliser pour capturer une idée, une " +
			"décision ou le résumé d'un échange que l'utilisateur voudra retrouver plus tard.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args addNoteArgs) (*mcp.CallToolResult, core.Note, error) {
		title := strings.TrimSpace(args.Title)
		if title == "" {
			return nil, core.Note{}, errors.New("title is required and cannot be empty")
		}

		callCtx, cancel := context.WithTimeout(ctx, apiCallTimeout)
		defer cancel()

		note, err := client.CreateNote(callCtx, title, args.Content, args.Tags)
		if err != nil {
			logger.Error("add_note: api call failed", "title", title, "error", err)
			return nil, core.Note{}, describeAPIError(err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "Note créée.\n\n" + formatNote(note)}},
		}, note, nil
	})
}

type listRecentArgs struct {
	Limit int `json:"limit,omitempty" jsonschema:"Nombre de notes récentes à retourner. Par défaut 10, plafonné à 100."`
}

func registerListRecentNotes(server *mcp.Server, client *apiclient.Client, logger *slog.Logger) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "list_recent_notes",
		Description: "Liste les notes créées le plus récemment, de la plus ancienne à la plus récente. " +
			"À utiliser pour donner un aperçu de l'activité récente de l'utilisateur, sans critère de " +
			"recherche particulier.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args listRecentArgs) (*mcp.CallToolResult, notesListOutput, error) {
		limit, err := normalizeLimit(args.Limit, defaultRecentLimit, maxRecentLimit)
		if err != nil {
			return nil, notesListOutput{}, err
		}

		callCtx, cancel := context.WithTimeout(ctx, apiCallTimeout)
		defer cancel()

		notes, err := client.ListRecent(callCtx, limit)
		if err != nil {
			logger.Error("list_recent_notes: api call failed", "limit", limit, "error", err)
			return nil, notesListOutput{}, describeAPIError(err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: formatNotesList(notes)}},
		}, notesListOutput{Notes: notes, Count: len(notes)}, nil
	})
}

// normalizeLimit applies the "optional limit with a default" convention
// used by all four tools: absent/zero falls back to def, negative is a
// validation error, and anything above max is silently capped.
func normalizeLimit(limit, def, max int) (int, error) {
	switch {
	case limit < 0:
		return 0, errors.New("limit must be a positive integer")
	case limit == 0:
		return def, nil
	case limit > max:
		return max, nil
	default:
		return limit, nil
	}
}

// describeAPIError turns an apiclient error into a clean, user-facing tool
// error: the mira API's own error code/message when available, or a generic
// connectivity message otherwise. The full error is always logged to stderr
// separately, so nothing here needs to leak internal detail.
func describeAPIError(err error) error {
	var apiErr *apiclient.APIError
	if errors.As(err, &apiErr) {
		return fmt.Errorf("mira API: %s", apiErr.Message)
	}
	return errors.New("could not reach the mira API, check that it is running and MIRA_API_URL is correct")
}

func formatNote(n core.Note) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n", n.Title)
	fmt.Fprintf(&b, "id: %s\n", n.ID)
	fmt.Fprintf(&b, "status: %s | enrichment: %s\n", n.Status, n.EnrichmentStatus)
	if len(n.Tags) > 0 {
		fmt.Fprintf(&b, "tags: %s\n", strings.Join(n.Tags, ", "))
	}
	if n.Summary != "" {
		fmt.Fprintf(&b, "summary: %s\n", n.Summary)
	}
	if n.Score != nil {
		fmt.Fprintf(&b, "score: %.3f\n", *n.Score)
	}
	fmt.Fprintf(&b, "created: %s | updated: %s\n", n.CreatedAt.Format(time.RFC3339), n.UpdatedAt.Format(time.RFC3339))
	if n.Content != "" {
		fmt.Fprintf(&b, "\n%s\n", n.Content)
	}
	return b.String()
}

func formatNotesList(notes []core.Note) string {
	if len(notes) == 0 {
		return "Aucune note trouvée."
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d note(s) :\n", len(notes))
	for _, n := range notes {
		fmt.Fprintf(&b, "- [%s] %s (status=%s, enrichment=%s", n.ID, n.Title, n.Status, n.EnrichmentStatus)
		if len(n.Tags) > 0 {
			fmt.Fprintf(&b, ", tags=%s", strings.Join(n.Tags, ", "))
		}
		b.WriteString(")\n")
	}
	return b.String()
}
