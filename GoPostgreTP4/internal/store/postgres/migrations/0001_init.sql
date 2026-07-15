-- Schéma initial : notes, tags, embeddings, index de recherche.

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE notes (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title              TEXT NOT NULL,
    content            TEXT NOT NULL DEFAULT '',
    status             TEXT NOT NULL DEFAULT 'active',
    enrichment_status  TEXT NOT NULL DEFAULT 'pending',
    summary            TEXT NOT NULL DEFAULT '',
    score              DOUBLE PRECISION,
    search_vector      TSVECTOR GENERATED ALWAYS AS (
                           setweight(to_tsvector('french', coalesce(title, '')), 'A') ||
                           setweight(to_tsvector('french', coalesce(content, '')), 'B')
                       ) STORED,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT notes_status_check CHECK (status IN ('active', 'archived')),
    CONSTRAINT notes_enrichment_status_check CHECK (enrichment_status IN ('pending', 'done', 'failed'))
);

CREATE TABLE note_tags (
    note_id  UUID NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    tag      TEXT NOT NULL,
    PRIMARY KEY (note_id, tag)
);

-- Dimension 64 : suffisant pour l'embedding local déterministe généré par
-- internal/enrichment (pas de dépendance à un modèle externe), tout en
-- restant représentatif d'un vrai pipeline pgvector.
CREATE TABLE note_embeddings (
    note_id     UUID PRIMARY KEY REFERENCES notes(id) ON DELETE CASCADE,
    embedding   VECTOR(64) NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notes_status ON notes (status);
CREATE INDEX idx_notes_search_vector ON notes USING GIN (search_vector);
CREATE INDEX idx_note_tags_tag ON note_tags (tag);
CREATE INDEX idx_note_embeddings_embedding ON note_embeddings USING hnsw (embedding vector_cosine_ops);