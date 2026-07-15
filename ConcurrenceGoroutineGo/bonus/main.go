package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

func worker(ctx context.Context, id int, jobs <-chan int, resultats chan<- int, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case j, ok := <-jobs:
			if !ok {
				return
			}
			// Lenteur liée à la valeur du job (1 job sur 4) : voir la
			// remarque dans ex5/ex5.go sur la répartition non déterministe
			// des jobs entre workers.
			if j%4 == 0 {
				time.Sleep(2 * time.Second)
			}
			select {
			case resultats <- j * j:
			case <-ctx.Done():
				return
			}
		}
	}
}

func main() {
	const nbJobs = 20
	const nbWorkers = 4

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	jobs := make(chan int, nbJobs)
	resultats := make(chan int, nbJobs)
	var wg sync.WaitGroup

	for w := 1; w <= nbWorkers; w++ {
		wg.Add(1)
		go worker(ctx, w, jobs, resultats, &wg)
	}

	for j := 1; j <= nbJobs; j++ {
		jobs <- j
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(resultats)
	}()

boucle:
	for {
		select {
		case r, ok := <-resultats:
			if !ok {
				break boucle
			}
			fmt.Println(r)
		case <-ctx.Done():
			fmt.Println("délai dépassé, annulation des workers restants :", ctx.Err())
			break boucle
		}
	}
}
