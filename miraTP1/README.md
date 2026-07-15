# TP #1 - Mira - CLI locale

Vous écrivez la base locale de **`mira`**, un outil de mémoire personnelle :

- stockage en **JSON Lines** (1 note par ligne dans **`~/.mira/notes.jsonl`**)
- commande **`mira add "titre" "contenu"`** pour créer une note
- commande **`mira list`** pour afficher les 10 dernières notes
- commande **`mira search <query>`** (recherche texte simple) pour retrouver une info

Structure attendue :

```bash
mira/
├── main.go          # point d'entrée CLI
└── internal/
    ├── notes/
    │   ├── note.go           # struct Note + constructeur
    │   ├── store.go          # interface NoteStore
    │   └── jsonl.go          # implémentation JSONL
    └── search/
        └── search.go         # recherche naïve sur titre+contenu
```

## Build

```bash
go build -o mira.exe .
```

## Tester

```bash
# Ajouter une note
./mira.exe add "Idée TP Go" "Écrire une CLI de mémoire personnelle en JSONL"
./mira.exe add "Recette pâtes" "Ail, huile d'olive, parmesan"

# Afficher les 10 dernières notes
./mira.exe list

# Rechercher un mot-clé dans le titre ou le contenu
./mira.exe search "go"
```

Les notes sont stockées dans `~/.mira/notes.jsonl` (une note JSON par ligne).

