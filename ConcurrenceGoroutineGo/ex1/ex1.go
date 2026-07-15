package main

import (
	"fmt"
	"time"
)

func afficherLettres() {
	for c := 'a'; c <= 'e'; c++ {
		fmt.Printf("%c\n", c)
		time.Sleep(50 * time.Millisecond)
	}
}

func afficherChiffres() {
	for i := 1; i <= 5; i++ {
		fmt.Println(i)
		time.Sleep(50 * time.Millisecond)
	}
}

func main() {
	go afficherLettres()
	afficherChiffres()

	// Sans ce Sleep, main() peut se terminer avant que la goroutine
	// afficherLettres() ait fini (ou même commencé) : le programme
	// s'arrête dès que main() retourne, sans attendre les goroutines.
	time.Sleep(300 * time.Millisecond)
}
