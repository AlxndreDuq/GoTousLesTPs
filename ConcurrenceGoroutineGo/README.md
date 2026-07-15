# TP #3 — Concurrence et goroutines en Go

## Objectifs

- Lancer et synchroniser des goroutines
- Utiliser des channels pour faire communiquer des goroutines
- Mettre en place un pattern worker pool
- Identifier et corriger une race condition

## Prérequis

- Go installé (`go version`)
- Un dossier de travail avec un `go.mod` :

```bash
mkdir tp-goroutines && cd tp-goroutines
go mod init tp-goroutines
```

---

## Exercice 1 — Première goroutine

Créez un fichier `ex1.go`.

Écrivez une fonction `afficherLettres()` qui affiche les lettres `a` à `e` (avec une pause de 50 ms entre chaque) et une fonction `afficherChiffres()` qui affiche les chiffres `1` à `5` (même pause).

Lancez `afficherLettres` en goroutine et `afficherChiffres` dans la goroutine principale.

**Question** : que se passe-t-il si vous retirez le `time.Sleep` final dans `main` ?

**Réponse** : `main` ne sait pas quand la goroutine `afficherLettres` a fini son travail ; dès que `afficherChiffres` (exécutée directement dans `main`) se termine, le programme s'arrête immédiatement, sans attendre la goroutine. Résultat : les lettres n'ont souvent pas le temps de s'afficher, ou seulement partiellement — le comportement devient imprévisible d'une exécution à l'autre.

---

## Exercice 2 — Synchronisation propre avec WaitGroup

Remplacez le `time.Sleep` de l'exercice 1 par un `sync.WaitGroup`.

Contraintes :

- `afficherLettres` et `afficherChiffres` doivent toutes les deux être lancées en goroutine
- `main` doit attendre la fin des deux sans utiliser `time.Sleep`

Squelette :

```go
package main

import (
	"fmt"
	"sync"
)

func afficherLettres(wg *sync.WaitGroup) {
	defer wg.Done()
	// TODO
}

func afficherChiffres(wg *sync.WaitGroup) {
	defer wg.Done()
	// TODO
}

func main() {
	var wg sync.WaitGroup
	// TODO : Add, go, go, Wait
}
```

---

## Exercice 3 — Somme parallèle avec channels

Écrivez un programme qui calcule la somme d'un slice de 1000 entiers en le divisant en 4 morceaux, chacun sommé par une goroutine différente, puis additionne les 4 résultats partiels reçus via un channel.

Indications :

- `make(chan int)` pour récupérer les résultats partiels
- une boucle de réception `a := <-resultat` répétée 4 fois (ou `for i := 0; i < 4; i++`)

Vérifiez que le résultat correspond à `n*(n+1)/2` pour n = 1000.

---

## Exercice 4 — Worker pool

Construisez un worker pool qui traite 20 "jobs" (des entiers de 1 à 20) avec 4 workers. Chaque worker doit :

- lire un entier depuis un channel `jobs`
- calculer son carré
- écrire le résultat dans un channel `resultats`

Contraintes :

- fermez `jobs` une fois tous les jobs envoyés
- utilisez un `sync.WaitGroup` pour savoir quand tous les workers ont terminé, puis fermez `resultats`
- affichez les résultats reçus (l'ordre ne sera pas forcément 1, 2, 3...)

**Question** : pourquoi l'ordre des résultats n'est-il pas garanti ? Répondez en commentaire dans le code.

**Réponse** : les 4 workers tournent en parallèle et lisent les jobs depuis le channel `jobs` dès qu'ils sont disponibles. Le scheduler Go décide de l'ordre d'exécution des goroutines, qui dépend du temps de traitement de chaque job et n'est pas déterministe : rien ne garantit qu'un worker traite les jobs dans l'ordre où ils ont été envoyés, ni que les résultats arrivent dans le channel `resultats` dans l'ordre des jobs d'origine.

---

## Exercice 5 — `select` et timeout

Modifiez le worker pool de l'exercice 4 (ou créez un nouveau fichier) pour simuler un traitement lent : un worker sur quatre attend 2 secondes avant de renvoyer son résultat (`time.Sleep`).

Dans `main`, utilisez un `select` avec un `case <-time.After(...)` pour abandonner l'attente d'un résultat au bout de 500 ms et afficher `"timeout sur un résultat"` le cas échéant.

---

## Exercice 6 — Trouver et corriger une race condition

Le code suivant contient une race condition. Copiez-le tel quel, exécutez-le avec `go run -race main.go`, puis corrigez-le avec un `sync.Mutex`.

```go
package main

import (
	"fmt"
	"sync"
)

func main() {
	compteur := 0
	var wg sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			compteur++
		}()
	}

	wg.Wait()
	fmt.Println("Compteur final :", compteur)
}
```

Questions :

1. Quel résultat obtenez-vous sans correction (exécutez plusieurs fois) ?
2. Que rapporte `go run -race main.go` ?
3. Après correction avec `sync.Mutex`, le résultat est-il stable ?

### Réponses

1. Sans correction, le résultat varie d'une exécution à l'autre et est presque toujours inférieur à 1000 (exemple observé sur 3 exécutions : `985`, `978`, `977`). C'est dû au fait que `compteur++` n'est pas une opération atomique : elle se décompose en une lecture, un incrément puis une écriture. Deux goroutines peuvent lire la même valeur avant que l'une des deux n'ait écrit la sienne, ce qui fait perdre des incréments.
2. `go run -race main.go` rapporte un **DATA RACE** sur la variable `compteur` : le rapport indique une écriture concurrente (`compteur++`) en conflit avec une autre écriture, accompagné des piles d'appel (`goroutine X` / `goroutine Y`) montrant qu'aucune synchronisation ne protège l'accès à la variable partagée.
3. Après correction avec un `sync.Mutex` (verrouillage autour de `compteur++`), le résultat est stable et vaut toujours exactement `1000`, quel que soit le nombre d'exécutions.

---

## Bonus — `context.Context`

Ajoutez un `context.WithTimeout` de 1 seconde à l'exercice 5 pour annuler proprement tous les workers restants si le traitement dépasse ce délai, plutôt que d'utiliser uniquement `select` côté `main`.