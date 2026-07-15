package main

import (
	"fmt"
	"os"
	"sort"
)

func main() {
	// Récupérer les tags en arguments
	tags := os.Args[1:]

	if len(tags) == 0 {
		fmt.Println("Error: no tags provided")
		os.Exit(1)
	}

	// Compter les occurrences de chaque tag
	tagCount := make(map[string]int)
	for _, tag := range tags {
		tagCount[tag]++
	}

	// Struct anonyme pour trier
	type tagCountPair struct {
		tag   string
		count int
	}

	// Construire une slice de tagCountPair
	var tagCountSlice []tagCountPair
	for tag, count := range tagCount {
		tagCountSlice = append(tagCountSlice, tagCountPair{tag, count})
	}

	// Trier par count décroissant (puis par tag alphabétiquement en cas d'égalité)
	sort.Slice(tagCountSlice, func(i, j int) bool {
		if tagCountSlice[i].count != tagCountSlice[j].count {
			return tagCountSlice[i].count > tagCountSlice[j].count
		}
		return tagCountSlice[i].tag < tagCountSlice[j].tag
	})

	// Afficher les tags avec count > 1
	fmt.Println("Tags appearing more than once:")
	for _, pair := range tagCountSlice {
		if pair.count > 1 {
			fmt.Printf("  %s: %d\n", pair.tag, pair.count)
		}
	}
}
