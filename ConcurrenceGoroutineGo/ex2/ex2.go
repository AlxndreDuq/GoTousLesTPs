package main

import (
	"fmt"
	"sync"
	"time"
)

func afficherLettres(wg *sync.WaitGroup) {
	defer wg.Done()
	for c := 'a'; c <= 'e'; c++ {
		fmt.Printf("%c\n", c)
		time.Sleep(50 * time.Millisecond)
	}
}

func afficherChiffres(wg *sync.WaitGroup) {
	defer wg.Done()
	for i := 1; i <= 5; i++ {
		fmt.Println(i)
		time.Sleep(50 * time.Millisecond)
	}
}

func main() {
	var wg sync.WaitGroup
	wg.Add(2)
	go afficherLettres(&wg)
	go afficherChiffres(&wg)
	wg.Wait()
}
