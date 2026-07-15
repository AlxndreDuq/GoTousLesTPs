# TP #2 - Mira - API v1

Mettre en place l'API /api/v1/notes du projet fil rouge.

**Endpoints attendus :**

- POST `/api/v1/notes` — créer une note
- GET `/api/v1/notes` — lister
- GET `/api/v1/notes/{id}` — récupérer
- PATCH `/api/v1/notes/{id}` — mettre à jour partiellement
- DELETE `/api/v1/notes/{id}` — supprimer
- GET `/api/v1/search?q=...` — recherche texte simple

Stockage en mémoire pour l'instant (map + mutex).

**Critères techniques :**

- Structure du repo **`cmd/api`** + **`internal/core`** + **`internal/http/handlers`** + **`internal/store`**
- Middlewares : request ID, logging structuré **`slog`**, recovery, timeout
- Validation explicite des payloads
- Enveloppe de réponse JSON stable
- Status HTTP corrects sur tous les chemins (succès et erreurs)
- README à jour : routes, exemples **`curl`**, codes d'erreur possibles

**Tests minimaux :**

- 1 test handler sur création (succès + 400)
- 1 test handler sur GET inexistant (404)

**Bonus :**

- pagination **`?limit=&offset=`** sur le LIST
- filtre **`?status=`**
- génération d'une spec OpenAPI à la main ou via outil

---

## Lancer l'API

```bash
go run ./cmd/api
```

Le serveur écoute par défaut sur le port `8080`. Le port est configurable via la variable d'environnement `PORT` :

```bash
PORT=9090 go run ./cmd/api
```

## Lancer les tests

```bash
go test ./...
```

## Architecture

```
cmd/api                    point d'entrée : wiring + arrêt propre (graceful shutdown)
internal/core              domaine : Note, règles de validation, Service (logique métier)
internal/store             persistance en mémoire (map + sync.RWMutex), implémente core.Repository
internal/http               routeur (net/http.ServeMux) + chaîne de middlewares
internal/http/middleware    request ID, logging (slog), recovery, timeout
internal/http/handlers      handlers HTTP, enveloppe de réponse JSON
```

La couche `core` ne dépend d'aucune bibliothèque HTTP : elle définit une interface `Repository`
implémentée par `internal/store`, ce qui permet de tester le service indépendamment du transport.

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
  "id": "1",
  "title": "Groceries",
  "content": "Milk, eggs",
  "status": "active",
  "created_at": "2026-07-13T10:00:00Z",
  "updated_at": "2026-07-13T10:00:00Z"
}
```

- `title` : requis, non vide (après trim), 200 caractères maximum.
- `content` : optionnel, 10 000 caractères maximum.
- `status` : optionnel, `"active"` (défaut) ou `"archived"`.

## Routes et exemples `curl`

### Créer une note

```bash
curl -i -X POST http://localhost:8080/api/v1/notes \
  -H "Content-Type: application/json" \
  -d '{"title":"Groceries","content":"Milk, eggs"}'
```

→ `201 Created`, header `Location: /api/v1/notes/{id}`, corps `{"data": {...}}`.

### Lister les notes

```bash
curl -i http://localhost:8080/api/v1/notes
```

Pagination (bonus) :

```bash
curl -i "http://localhost:8080/api/v1/notes?limit=10&offset=0"
```

Filtre par statut (bonus) :

```bash
curl -i "http://localhost:8080/api/v1/notes?status=archived"
```

→ `200 OK`, corps `{"data": [...], "meta": {"total": N, "limit": 20, "offset": 0}}`.
`limit` par défaut 20, plafonné à 100. `limit`/`offset` négatifs ou non numériques → `400`.

### Récupérer une note

```bash
curl -i http://localhost:8080/api/v1/notes/1
```

→ `200 OK` ou `404 Not Found`.

### Mettre à jour partiellement une note

```bash
curl -i -X PATCH http://localhost:8080/api/v1/notes/1 \
  -H "Content-Type: application/json" \
  -d '{"status":"archived"}'
```

Seuls les champs présents dans le corps sont modifiés. → `200 OK`, `400` (validation) ou `404`.

### Supprimer une note

```bash
curl -i -X DELETE http://localhost:8080/api/v1/notes/1
```

→ `204 No Content` ou `404 Not Found`.

### Recherche texte

```bash
curl -i "http://localhost:8080/api/v1/search?q=milk"
```

Recherche insensible à la casse sur `title` et `content`. `q` manquant/vide → `400`.
→ `200 OK`, corps `{"data": [...], "meta": {"total": N}}`.

## Codes d'erreur

| Status | `error.code`        | Cas                                                              |
|--------|---------------------|-------------------------------------------------------------------|
| 400    | `invalid_json`       | corps de requête absent ou JSON invalide                         |
| 400    | `validation_error`   | payload invalide (titre manquant, statut inconnu, pagination négative, `q` manquant, ...) |
| 404    | `not_found`          | note inexistante (GET/PATCH/DELETE)                               |
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

## Bonus implémentés

- Pagination `?limit=&offset=` sur le LIST.
- Filtre `?status=` sur le LIST.
- Spec OpenAPI écrite à la main : [`openapi.yaml`](openapi.yaml).
- Documentation interactive : `GET /docs` sert une page Swagger UI (assets chargés depuis un CDN, servie par l'API elle-même via `go:embed`), qui lit la spec exposée sur `GET /docs/openapi.yaml`. Accessible directement sur `http://localhost:8080/docs`, sans outil externe à lancer. Nécessite de lancer le serveur depuis la racine du repo (`openapi.yaml` est lu depuis le disque).
