package main

import (
	"fmt"
	"sync"
	"time"
)

func worker(id int, jobs <-chan int, resultats chan<- int, wg *sync.WaitGroup) {
	defer wg.Done()
	for j := range jobs {
		// La lenteur est liée à la valeur du job (1 job sur 4) plutôt qu'à
		// l'identité du worker : comme les jobs sont répartis entre workers
		// selon l'ordre d'arrivée (non déterministe), lier la lenteur au
		// worker ne garantirait pas de façon fiable qu'un traitement lent
		// se produise à chaque exécution.
		if j%4 == 0 {
			time.Sleep(2 * time.Second)
		}
		resultats <- j * j
	}
}

func main() {
	const nbJobs = 20
	const nbWorkers = 4

	jobs := make(chan int, nbJobs)
	resultats := make(chan int, nbJobs)
	var wg sync.WaitGroup

	for w := 1; w <= nbWorkers; w++ {
		wg.Add(1)
		go worker(w, jobs, resultats, &wg)
	}

	for j := 1; j <= nbJobs; j++ {
		jobs <- j
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(resultats)
	}()

	for {
		select {
		case r, ok := <-resultats:
			if !ok {
				return
			}
			fmt.Println(r)
		case <-time.After(500 * time.Millisecond):
			fmt.Println("timeout sur un résultat")
		}
	}
}
