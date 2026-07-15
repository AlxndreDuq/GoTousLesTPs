package main

import (
	"fmt"
	"sync"
)

func main() {
	compteur := 0
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			compteur++
			mu.Unlock()
		}()
	}

	wg.Wait()
	fmt.Println("Compteur final :", compteur)
}
