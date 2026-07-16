
# Démarrer

### 1. Base de données

```bash
docker compose up -d
```

Lance PostgreSQL 16 + l'extension **pgvector** (image `pgvector/pgvector:pg16`), sur `localhost:5432`
(utilisateur/mot de passe/base : `mira`/`mira`/`mira`). Les migrations SQL sont appliquées
automatiquement au démarrage de l'API (pas d'étape manuelle).

### 2. API

```bash
go run ./cmd/api
```

Le serveur écoute par défaut sur le port `8080`. Configuration via variables d'environnement
(voir [`.env.example`](.env.example)) :

| Variable                 | Défaut                                                        | Rôle                                     |
|---------------------------|----------------------------------------------------------------|-------------------------------------------|
| `PORT`                    | `8080`                                                          | Port HTTP                                 |
| `DATABASE_URL`            | `postgres://mira:mira@localhost:5432/mira?sslmode=disable`     | Connexion PostgreSQL                      |
| `ENRICHMENT_WORKERS`      | `4`                                                             | Taille du pool de workers d'enrichissement |
| `ENRICHMENT_QUEUE_SIZE`   | `100`                                                           | Taille du buffer du channel de jobs       |
| `ENRICHMENT_TIMEOUT`      | `10s`                                                           | Timeout par tâche d'enrichissement         |

### 3. CLI

```bash
go build -o mira.exe ./cmd/mira

./mira.exe add "Idée TP Go" "Écrire une API sur PostgreSQL avec enrichissement asynchrone"
./mira.exe list
./mira.exe search "PostgreSQL"
```

La CLI parle exclusivement à l'API HTTP (`MIRA_API_URL`, défaut `http://localhost:8080`) —
elle ne touche plus de fichier local. `add` déclenche donc bien l'enrichissement automatique côté
serveur, comme n'importe quel autre client de l'API.

### 4. Serveur MCP

```bash
go build -o mira-mcp.exe ./cmd/mira-mcp
```

`cmd/mira-mcp` expose la mémoire de `mira` à un agent IA (Claude Code, Claude Desktop, ...) via le
**Model Context Protocol**, en transport **stdio**. Comme la CLI, il parle exclusivement à l'API
HTTP (`MIRA_API_URL`, défaut `http://localhost:8080`) — jamais à la base en direct — pour garantir
que les notes créées par un agent déclenchent bien l'enrichissement automatique. Voir
[Serveur MCP](#serveur-mcp-cmdmira-mcp) plus bas pour l'enregistrement dans Claude Code/Desktop et
des exemples de prompts.

## Lancer les tests

```bash
go test ./...
```

Les tests unitaires (`internal/core`, `internal/embedding`, `internal/enrichment`,
`internal/http/handlers`) tournent sans dépendance externe. Les tests d'intégration
(`internal/store/postgres`) se **skip automatiquement** si `DATABASE_URL` n'est pas joignable ;
avec `docker compose up -d` lancé au préalable, ils s'exécutent pour de vrai contre PostgreSQL
(transactions, cast UUID, merge de tags, upsert d'embedding, recherche full-text).

## Architecture

```
cmd/api                     point d'entrée API : wiring, migrations, pool pgx, dispatcher +
                             worker pool d'enrichissement, arrêt propre (graceful shutdown)
cmd/mira                    CLI (add/list/search), parle à l'API en HTTP
cmd/mira-mcp                 serveur MCP (transport stdio) exposant search_notes/get_note/
                             add_note/list_recent_notes à un agent, parle à l'API en HTTP

internal/core                domaine : Note, règles de validation, Service, interfaces
                             Repository et EnrichmentQueue
internal/embedding           fonction pure texte -> vecteur pseudo-aléatoire déterministe
                             (pas de dépendance externe / clé API)
internal/enrichment           pipeline d'enrichissement : Job, Dispatcher (channel interne),
                             Pool (workers bornés), Enricher (NaiveEnricher)
internal/store/postgres       repository PostgreSQL (pgx), migrations embarquées (go:embed),
                             recherche hybride full-text + vectorielle (pgvector)
internal/apiclient            client HTTP utilisé par cmd/mira
internal/http                 routeur (net/http.ServeMux) + chaîne de middlewares
internal/http/middleware      request ID, logging (slog), recovery, timeout
internal/http/handlers        handlers HTTP, enveloppe de réponse JSON

migrations/*.sql (embarquées dans internal/store/postgres/migrations)
```

La couche `core` ne dépend d'aucune bibliothèque HTTP ni PostgreSQL : elle définit les interfaces
`Repository` (persistance) et `EnrichmentQueue` (publication de jobs), implémentées respectivement
par `internal/store/postgres` et `internal/enrichment`. Cela permet de tester `Service` avec de
faux repository/queue en mémoire (voir `internal/core/service_test.go` et
`internal/http/handlers/notes_test.go`), sans base de données.

## Pipeline d'enrichissement

```
POST/PATCH note ──(écriture DB synchrone)──> réponse HTTP immédiate
        │
        └─(note.ID)──> Dispatcher.Enqueue ──channel (bufferisé)──> Pool de N workers
                                                                       │
                                                     Get(id) ─> Enrich(ctx, note) ─> SaveEnrichment
                                                     (timeout ENRICHMENT_TIMEOUT par tâche)
```

- **Dispatcher** (`internal/enrichment/dispatcher.go`) : `Enqueue` est **non bloquant** — si le
  channel est plein, le job est abandonné et loggé (`enrichment_queue_full`) plutôt que de
  ralentir la réponse HTTP. La note reste alors en `enrichment_status = "pending"`.
- **Pool** (`internal/enrichment/pool.go`) : `ENRICHMENT_WORKERS` goroutines consomment le même
  channel. Chaque job récupère la note à jour, l'enrichit avec un `context.WithTimeout` propre
  (indépendant du contexte de la requête HTTP d'origine), puis persiste le résultat.
- **NaiveEnricher** (`internal/enrichment/naive.go`) : génère tags (mots-clés les plus fréquents,
  hors mots vides), résumé (première phrase), score (heuristique longueur + richesse en tags) et
  embedding (`internal/embedding`, vecteur déterministe de dimension 64, sans appel externe).
- **Tags additifs** : les tags fournis à la création (transaction note + tags) et les tags générés
  par l'enrichissement se **complètent** (`INSERT ... ON CONFLICT DO NOTHING`), l'un n'écrase
  jamais l'autre. Un `PATCH` qui fournit explicitement `tags` **remplace** en revanche la liste
  complète (c'est un choix explicite du client, pas une inférence automatique).
- Modifier `title` ou `content` via `PATCH` republie un job (l'ancien résumé/tags/embedding sont
  périmés) et remet `enrichment_status` à `pending` ; un `PATCH` qui ne touche que `status` ou
  `tags` ne redéclenche rien.
- **Arrêt propre** : à la réception de SIGINT/SIGTERM, `cmd/api` arrête d'abord le serveur HTTP
  (plus aucun nouveau job possible), ferme le dispatcher, puis attend que les workers vident le
  buffer restant (borné par la même fenêtre de 10s que le `Shutdown` HTTP).

## Recherche hybride

`GET /api/v1/search?q=...` combine, en une seule requête SQL :

- un score **full-text** (`ts_rank` sur un `tsvector` généré en base, indexé en **GIN**, dictionnaire
  `french`) ;
- une similarité **cosinus** entre l'embedding de la requête (calculé avec la même fonction
  déterministe que l'enrichissement) et `note_embeddings.embedding`, indexé en **HNSW** (`pgvector`).

Les deux scores sont combinés par une somme pondérée simple (`0.6 * fts_rank + 0.4 * vec_sim`) —
une note pas encore enrichie peut malgré tout ressortir via son seul score full-text. C'est une
heuristique volontairement simple : `ts_rank` et la similarité cosinus vivent sur des échelles
différentes, une vraie fusion calibrerait chaque signal (min-max, reciprocal rank fusion...) plutôt
que d'utiliser des poids fixes. L'embedding lui-même est un hachage déterministe du texte (pas un
modèle sémantique) : la similarité vectorielle capture surtout un recouvrement de vocabulaire, pas
un vrai rapprochement de sens — suffisant pour exercer le pipeline `pgvector` de bout en bout sans
dépendance à une API externe/clé.

## Enveloppe de réponse

Toutes les réponses JSON suivent la même forme stable :

Succès :

```json
{ "data": { "...": "..." } }
```

Succès (liste, avec métadonnées de pagination) :

```json
{ "data": [ { "...": "..." } ], "meta": { "total": 3, "limit": 20, "offset": 0 } }
```

Erreur :

```json
{ "error": { "code": "validation_error", "message": "title is required" } }
```

## Modèle de note

```json
{
  "id": "bbfd4a70-dfee-4b71-a44f-5eb7f63c7b5e",
  "title": "Recette pâtes carbonara",
  "content": "Lardons, œufs, parmesan",
  "status": "active",
  "tags": ["maison", "carbonara", "recette"],
  "enrichment_status": "done",
  "summary": "Faire cuire les pâtes al dente.",
  "score": 0.586,
  "created_at": "2026-07-15T11:04:07.808534+02:00",
  "updated_at": "2026-07-15T11:04:07.813822+02:00"
}
```

- `title` : requis, non vide (après trim), 200 caractères maximum.
- `content` : optionnel, 10 000 caractères maximum.
- `status` : optionnel, `"active"` (défaut) ou `"archived"`.
- `tags` : optionnel à la création, 20 tags maximum, 50 caractères chacun.
- `enrichment_status` : géré par le serveur (`pending` / `done` / `failed`), lecture seule.
- `summary`, `score` : produits par l'enrichissement, vides/nuls tant que `pending`.

## Routes et exemples `curl`

### Créer une note

```bash
curl -i -X POST http://localhost:8080/api/v1/notes \
  -H "Content-Type: application/json" \
  -d '{"title":"Groceries","content":"Milk, eggs","tags":["maison"]}'
```

→ `201 Created`, header `Location: /api/v1/notes/{id}`, corps `{"data": {...}}` avec
`enrichment_status: "pending"`. Quelques centaines de millisecondes plus tard (selon la charge du
pool de workers), un `GET` sur la même note affichera `enrichment_status: "done"` et les
tags/résumé/score générés.

### Lister les notes

```bash
curl -i http://localhost:8080/api/v1/notes
```

Pagination : `?limit=10&offset=0` (défaut 20, plafonné à 100 ; valeurs négatives/non numériques →
`400`). Filtre par statut : `?status=archived`.

→ `200 OK`, corps `{"data": [...], "meta": {"total": N, "limit": 20, "offset": 0}}`.

### Récupérer une note

```bash
curl -i http://localhost:8080/api/v1/notes/{id}
```

→ `200 OK` ou `404 Not Found`.

### Mettre à jour partiellement une note

```bash
curl -i -X PATCH http://localhost:8080/api/v1/notes/{id} \
  -H "Content-Type: application/json" \
  -d '{"content":"Milk, eggs, butter"}'
```

Seuls les champs présents dans le corps sont modifiés. → `200 OK`, `400` (validation) ou `404`.
Modifier `title`/`content` republie un job d'enrichissement (voir plus haut).

### Supprimer une note

```bash
curl -i -X DELETE http://localhost:8080/api/v1/notes/{id}
```

→ `204 No Content` ou `404 Not Found`.

### Recherche hybride

```bash
curl -i "http://localhost:8080/api/v1/search?q=milk"
```

`q` manquant/vide → `400`. → `200 OK`, corps `{"data": [...], "meta": {"total": N}}`, trié par
score hybride décroissant (voir [Recherche hybride](#recherche-hybride)).

## Codes d'erreur

| Status | `error.code`        | Cas                                                              |
|--------|---------------------|-------------------------------------------------------------------|
| 400    | `invalid_json`       | corps de requête absent ou JSON invalide                         |
| 400    | `validation_error`   | payload invalide (titre manquant, statut inconnu, pagination négative, `q` manquant, trop de tags, ...) |
| 404    | `not_found`          | note inexistante (GET/PATCH/DELETE), y compris un `id` mal formé |
| 500    | `internal_error`     | erreur inattendue (détail loggé côté serveur, pas exposé au client) |
| 504    | `timeout`            | la requête dépasse le délai imparti (5s)                          |

Note : les routes/méthodes hors du contrat ci-dessus (ex. `POST /api/v1/notes/{id}`) retombent sur
le comportement par défaut de `net/http.ServeMux` (404/405 en texte brut, hors enveloppe JSON).

## Middlewares

Chaîne (de l'extérieur vers l'intérieur) : `RequestID` → `Logging` → `Recovery` → `Timeout` → routeur.

- **RequestID** : génère (ou réutilise) un identifiant de requête, exposé via le header `X-Request-ID`.
- **Logging** : une ligne de log structuré (`slog`, JSON) par requête — méthode, chemin, status, durée, request ID.
- **Recovery** : capture les paniques, log la stack, répond `500` au lieu de crasher.
- **Timeout** : borne chaque requête à 5s ; au-delà, répond `504` sans corrompre la connexion si le handler écrit encore.

## Documentation interactive

`GET /docs` sert une page Swagger UI (assets chargés depuis un CDN, servie par l'API elle-même via
`go:embed`), qui lit la spec exposée sur `GET /docs/openapi.yaml`. Nécessite de lancer le serveur
depuis la racine du repo (`openapi.yaml` est lu depuis le disque). Spec écrite à la main :
[`openapi.yaml`](openapi.yaml).

## Serveur MCP (`cmd/mira-mcp`)

`cmd/mira-mcp` branche la mémoire de `mira` sur un agent IA (Claude Code, Claude Desktop, tout
client [MCP](https://modelcontextprotocol.io)) via un serveur **MCP en transport stdio**
(JSON-RPC 2.0), construit avec le SDK officiel
[`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk). Comme la CLI, il
passe exclusivement par l'API HTTP de `mira` (`internal/apiclient`), jamais par la base en direct —
c'est ce qui garantit que la création d'une note par l'agent déclenche bien l'enrichissement
automatique côté serveur.

### Tools exposés

| Tool | Paramètres | Rôle |
| --- | --- | --- |
| `search_notes` | `query` (string, requis), `limit` (int, défaut 10, max 50) | recherche hybride full-text + vectorielle |
| `get_note` | `id` (string, requis) | note complète : contenu, tags, résumé, score, statut d'enrichissement |
| `add_note` | `title` (string, requis), `content` (string, requis), `tags` (array de strings, optionnel) | crée une note (déclenche l'enrichissement asynchrone côté API) |
| `list_recent_notes` | `limit` (int, défaut 10, max 100) | dernières notes créées, de la plus ancienne à la plus récente |

Chaque appel API sous-jacent est borné par un timeout `context` de 10s, indépendant du timeout
propre à l'agent appelant. Les erreurs (validation, note introuvable, API injoignable, ...) sont
renvoyées comme des erreurs MCP propres (`isError: true` + message) — jamais de panic ni de stack
trace brute. Aucun log n'est écrit sur stdout (réservé au protocole JSON-RPC) ; tous les logs
passent par `slog` sur **stderr**.

### Installation

```bash
go build -o mira-mcp.exe ./cmd/mira-mcp
```

L'URL de l'API est configurable via `MIRA_API_URL` (défaut `http://localhost:8080`, voir
[`.env.example`](.env.example)). L'API (`go run ./cmd/api`) et PostgreSQL (`docker compose up -d`)
doivent tourner pour que les tools fonctionnent.

### Enregistrement dans Claude Code

Un fichier d'exemple est fourni à la racine du repo : [`.mcp.json`](.mcp.json).

```json
{
  "mcpServers": {
    "mira": {
      "command": "go",
      "args": ["run", "./cmd/mira-mcp"],
      "env": {
        "MIRA_API_URL": "http://localhost:8080"
      }
    }
  }
}
```

Claude Code détecte automatiquement `.mcp.json` à la racine du projet ouvert. Pour utiliser le
binaire compilé plutôt que `go run` (démarrage plus rapide), remplacer `command`/`args` par
`"command": "./mira-mcp.exe"` (ou le chemin absolu vers le binaire) et retirer `args`.

Vous pouvez aussi enregistrer le serveur en ligne de commande, depuis la racine du repo :

```bash
claude mcp add mira -- go run ./cmd/mira-mcp
```

### Enregistrement dans Claude Desktop

Ajouter dans la configuration MCP de Claude Desktop (`claude_desktop_config.json`) :

```json
{
  "mcpServers": {
    "mira": {
      "command": "C:\\chemin\\vers\\mira-mcp.exe",
      "env": {
        "MIRA_API_URL": "http://localhost:8080"
      }
    }
  }
}
```

(Claude Desktop lance le process lui-même : préférer un chemin vers le binaire compilé plutôt que
`go run`, pour ne pas dépendre de l'environnement Go de l'utilisateur.)

### Exemples de prompts

Une fois le serveur enregistré et l'API lancée, dans Claude Code :

- *"Retrouve ma note sur les channels Go"* → appelle `search_notes`
- *"Ajoute une note résumant ce qu'on vient de faire"* → appelle `add_note`
- *"Montre-moi le contenu complet de la note bbfd4a70-..."* → appelle `get_note`
- *"Qu'est-ce que j'ai noté récemment ?"* → appelle `list_recent_notes`
