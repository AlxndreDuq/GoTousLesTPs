package main

import (
	"fmt"
	"os"
)

const MaxDisplay = 10

func main() {
	// os.Args[0] est le nom du programme, os.Args[1:] sont les arguments
	args := os.Args[1:]

	// Vérifier qu'au moins un argument est fourni
	if len(args) == 0 {
		fmt.Println("Error: no arguments provided")
		os.Exit(1)
	}

	// Afficher le nombre total de mots
	fmt.Printf("Total words: %d\n", len(args))

	// Afficher les mots de longueur > 4
	fmt.Println("Words with length > 4:")
	for _, word := range args {
		if len(word) > 4 {
			fmt.Printf("  - %s (length: %d)\n", word, len(word))
		}
	}
}
