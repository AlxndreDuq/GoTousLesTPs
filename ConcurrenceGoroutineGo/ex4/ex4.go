package main

import (
	"fmt"
	"sync"
)

func worker(id int, jobs <-chan int, resultats chan<- int, wg *sync.WaitGroup) {
	defer wg.Done()
	for j := range jobs {
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

	// L'ordre des résultats n'est pas garanti car les 4 workers tournent
	// en parallèle et lisent les jobs dès qu'ils sont disponibles : le
	// scheduler Go décide de l'ordre d'exécution, qui dépend du temps
	// de traitement de chaque job et n'est pas déterministe.
	go func() {
		wg.Wait()
		close(resultats)
	}()

	for r := range resultats {
		fmt.Println(r)
	}
}
