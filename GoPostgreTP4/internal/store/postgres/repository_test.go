package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"mira-tp4/internal/core"
	"mira-tp4/internal/store/postgres"
)

// connectTestPool opens a pool against DATABASE_URL (same default as the
// app) and skips the test if Postgres isn't reachable, so `go test ./...`
// stays green without Docker running.
func connectTestPool(t *testing.T) *postgres.Repository {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://mira:mira@localhost:5432/mira?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	pool, err := postgres.NewPool(ctx, databaseURL)
	if err != nil {
		t.Skipf("postgres not reachable at %s, skipping integration test: %v", databaseURL, err)
	}
	t.Cleanup(pool.Close)

	if err := postgres.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	return postgres.NewRepository(pool)
}

func TestRepository_CreateGetUpdateDelete(t *testing.T) {
	repo := connectTestPool(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, core.Note{
		Title:   "Recette pâtes carbonara",
		Content: "Lardons, œufs, parmesan",
		Status:  core.StatusActive,
		Tags:    []string{"cuisine", "italien"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(context.Background(), created.ID) })

	if created.ID == "" {
		t.Fatal("expected a generated id")
	}
	if created.EnrichmentStatus != core.EnrichmentPending {
		t.Fatalf("EnrichmentStatus = %q, want %q", created.EnrichmentStatus, core.EnrichmentPending)
	}
	if len(created.Tags) != 2 {
		t.Fatalf("Tags = %v, want 2 user-supplied tags", created.Tags)
	}

	fetched, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if fetched.Title != created.Title {
		t.Fatalf("Title = %q, want %q", fetched.Title, created.Title)
	}

	newTitle := "Recette pâtes carbonara (v2)"
	updated, err := repo.Update(ctx, created.ID, core.UpdateInput{Title: &newTitle})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Title != newTitle {
		t.Fatalf("Title = %q, want %q", updated.Title, newTitle)
	}
	if updated.EnrichmentStatus != core.EnrichmentPending {
		t.Fatalf("EnrichmentStatus after content-affecting update = %q, want %q (reset)", updated.EnrichmentStatus, core.EnrichmentPending)
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := repo.Get(ctx, created.ID); err != core.ErrNotFound {
		t.Fatalf("Get() after delete error = %v, want core.ErrNotFound", err)
	}
}

func TestRepository_Get_InvalidIDIsNotFound(t *testing.T) {
	repo := connectTestPool(t)

	_, err := repo.Get(context.Background(), "not-a-uuid")
	if err != core.ErrNotFound {
		t.Fatalf("error = %v, want core.ErrNotFound", err)
	}
}

func TestRepository_SaveEnrichment_MergesTagsAndUpsertsEmbedding(t *testing.T) {
	repo := connectTestPool(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, core.Note{
		Title:   "Voyage au Japon",
		Content: "Préparer un itinéraire pour Tokyo et Kyoto",
		Status:  core.StatusActive,
		Tags:    []string{"perso"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(context.Background(), created.ID) })

	embedding := make([]float32, 64)
	embedding[0] = 1

	err = repo.SaveEnrichment(ctx, core.EnrichmentResult{
		NoteID:    created.ID,
		Status:    core.EnrichmentDone,
		Tags:      []string{"voyage", "japon"},
		Summary:   "Itinéraire Japon.",
		Score:     0.8,
		Embedding: embedding,
	})
	if err != nil {
		t.Fatalf("SaveEnrichment() error = %v", err)
	}

	fetched, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if fetched.EnrichmentStatus != core.EnrichmentDone {
		t.Fatalf("EnrichmentStatus = %q, want %q", fetched.EnrichmentStatus, core.EnrichmentDone)
	}
	if fetched.Score == nil || *fetched.Score != 0.8 {
		t.Fatalf("Score = %v, want 0.8", fetched.Score)
	}

	wantTags := map[string]bool{"perso": false, "voyage": false, "japon": false}
	for _, tag := range fetched.Tags {
		if _, ok := wantTags[tag]; ok {
			wantTags[tag] = true
		}
	}
	for tag, found := range wantTags {
		if !found {
			t.Fatalf("expected tag %q to be present in %v (user tags must survive enrichment)", tag, fetched.Tags)
		}
	}
}

func TestRepository_Search_FindsFullTextMatch(t *testing.T) {
	repo := connectTestPool(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, core.Note{
		Title:   "Randonnée en montagne",
		Content: "Prévoir des chaussures de randonnée et de l'eau",
		Status:  core.StatusActive,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() { _ = repo.Delete(context.Background(), created.ID) })

	result, err := repo.Search(ctx, "randonnée")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	found := false
	for _, n := range result.Notes {
		if n.ID == created.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected note %q in search results %+v", created.ID, result.Notes)
	}
}
