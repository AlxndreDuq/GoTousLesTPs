# Go - Exercices d’échauffement

Avant de démarrer `mira`, on passe par une série de micro-exercices pour ancrer :

- la structure d'un programme Go,
- les types, slices, maps et boucles,
- les structs, méthodes et interfaces,
- les pointeurs et la gestion d'erreurs,
- le réflexe `go fmt` + `go test`.

Tous les exercices seront regroupés dans un dossier `go-warmup/`.

**Exercice 1 — Variables, types et boucles**

Écrire `go-warmup/ex1/main.go` qui :

- déclare une `const MaxDisplay = 10`,
- prend une liste de mots en `os.Args`,
- affiche le nombre total de mots et les mots de longueur > 4,
- retourne un code d'erreur si aucun argument n'est fourni.

Questions à explorer :

```go
// Que retourne ce code ?

var s string

fmt.Printf("%q\n", s)

// Quelle est la différence ?

x := 0

var y int
```

Notions : `os.Args`, `if`, `for range`, `len`, zero values, `os.Exit(1)`.

**Exercice 2 — Slices et maps : compter et trier**

Écrire `go-warmup/ex2/main.go` qui :

- lit une liste de tags passés en arguments (ex: `go api backend go rest go`),
- compte les occurrences de chaque tag avec une `map[string]int`,
- affiche les tags triés par fréquence décroissante,
- affiche uniquement les tags apparaissant plus d'une fois.

```go
// Aide : trier une map par valeur

import "sort"

type tagCount struct {

tag   string

count int

}

// ... construire un []tagCount depuis la map, puis sort.Slice(...)
```

Notions : `map`, `sort.Slice`, itération, `struct` anonyme.

**Exercice 3 — Structs, méthodes et defer**

Créer une struct `Note` avec `Title`, `Content`, `Tags`, puis :

- écrire une fonction `NewNote(title, content string) *Note`,
- écrire une méthode `Preview() string` qui renvoie les 80 premiers caractères du contenu,
- écrire une méthode `AddTag(tag string)` qui évite les doublons,
- écrire une fonction `LoadFromFile(path string) ([]*Note, error)` qui utilise `defer f.Close()`,
- afficher les notes dont au moins un tag correspond à `"go"`.

Notions : struct, méthode sur pointeur, defer, slice, comparaison de chaînes.

**Exercice 4 — Interfaces et erreurs typées**

Définir :

```go
var ErrDuplicate = errors.New("note already exists")

var ErrNotFound  = errors.New("note not found")

type NoteStore interface {
	Save(n *Note) error
	Get(id string) (*Note, error
	All() []*Note
}
```

Implémenter `MemoryStore` (struct avec `map[string]*Note + sync.Mutex`).

Contraintes :

- `Save` refuse une note sans titre (`ErrValidation`),
- `Save` refuse les doublons d'ID (`ErrDuplicate`),
- `Get` retourne `ErrNotFound` si l'ID est absent.

Notions : interface implicite, `errors.New`, `errors.Is`, mutation via pointeur, `sync.Mutex`.

**Exercice 5 — Tests unitaires**

Écrire `store_test.go` avec au moins 4 cas :

```go
func TestSave_valid(t *testing.T) { /* note valide → pas d'erreur */ }

func TestSave_emptyTitle(t *testing.T) { /* titre vide → erreur */ }

func TestSave_duplicate(t *testing.T) { /* double Save → ErrDuplicate */ }

func TestGet_notFound(t *testing.T) { /* Get("?") → ErrNotFound */ }
```

Commandes :

```bash
go fmt ./...

go vet ./...

go test ./... -v
```

> En Go, l'outillage fait partie du langage. `go test` est là depuis Go 1.0.