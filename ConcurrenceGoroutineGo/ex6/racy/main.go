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

// Questions :
// 1. Sans correction, le résultat varie d'une exécution à l'autre et est
//    presque toujours inférieur à 1000 (ex: 950, 987...), car compteur++
//    n'est pas atomique : il se décompose en lecture, incrémentation,
//    écriture, et deux goroutines peuvent lire la même valeur avant que
//    l'une des deux n'ait écrit son résultat, ce qui perd des incréments.
// 2. `go run -race main.go` rapporte un DATA RACE sur la variable
//    compteur, avec les piles d'appel des goroutines en conflit
//    (lecture concurrente et écriture concurrente sans synchronisation).
// 3. Voir ex6/fixed/main.go : avec un sync.Mutex, le résultat est stable
//    et vaut toujours exactement 1000.
