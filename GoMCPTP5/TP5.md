# TP #5 - Serveur MCP : brancher Mira sur les agents IA

### Contexte

Le **Model Context Protocol (MCP)** est le standard ouvert qui permet aux agents IA
(Claude Code, Claude Desktop, IDE, etc.) de découvrir et d'appeler des outils externes.

Objectif de ce TP : exposer la mémoire de `mira` à un agent, pour qu'il puisse
chercher, lire et créer des notes **pendant une conversation**.

À la fin du TP, vous devez pouvoir demander à Claude Code :
*"Retrouve ma note sur les channels Go et ajoute une note résumant ce qu'on vient de faire"* et voir les appels transiter par votre serveur.

### Brief

Créer un binaire `cmd/mira-mcp` qui :

- implémente un **serveur MCP en transport stdio** (JSON-RPC 2.0)
- utilise un SDK Go existant (`modelcontextprotocol/go-sdk`)
- expose 4 **tools** :

| Tool | Paramètres | Rôle |
| --- | --- | --- |
| `search_notes` | `query` (string, requis), `limit` (int, défaut 10) | recherche hybride full-text + vectorielle |
| `get_note` | `id` (string, requis) | retourne une note complète (contenu, tags, résumé, statut) |
| `add_note` | `title`, `content` (requis), `tags` (optionnel) | crée une note |
| `list_recent_notes` | `limit` (int, défaut 10) | dernières notes créées |

> ⚠️ Le serveur MCP passe **par l'API HTTP de mira** (comme la CLI), jamais par la base
en direct : c'est ce qui garantit que l'enrichissement automatique soit déclenché.
> 

### Critères techniques

- Chaque tool a une **description soignée** et un **schéma JSON strict** (l'agent choisit
ses outils en lisant vos descriptions : elles font partie du contrat)
- Validation des inputs + erreurs MCP propres (jamais de panic, jamais de stack trace brute)
- Timeout via `context` sur chaque appel API sous-jacent
- URL de l'API configurable (`MIRA_API_URL`)
- Aucun log sur **stdout** (réservé au protocole en transport stdio) → logs `slog` sur stderr
- Fichier de configuration d'exemple pour Claude Code (`.mcp.json`) fourni dans le repo
- README : installation, enregistrement du serveur dans Claude Code/Desktop, exemples de prompts

### Critères de validation

- démo live : un agent (Claude Code) appelle réellement `search_notes` et `add_note`
- la note créée par l'agent apparaît enrichie (`enrichment_status: done`) côté API