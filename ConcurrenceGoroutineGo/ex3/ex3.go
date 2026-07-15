package main

import "fmt"

const n = 1000
const nbMorceaux = 4

func sommePartielle(nombres []int, resultat chan<- int) {
	somme := 0
	for _, v := range nombres {
		somme += v
	}
	resultat <- somme
}

func main() {
	nombres := make([]int, n)
	for i := range nombres {
		nombres[i] = i + 1
	}

	tailleMorceau := n / nbMorceaux
	resultat := make(chan int)

	for i := 0; i < nbMorceaux; i++ {
		debut := i * tailleMorceau
		fin := debut + tailleMorceau
		go sommePartielle(nombres[debut:fin], resultat)
	}

	somme := 0
	for i := 0; i < nbMorceaux; i++ {
		somme += <-resultat
	}

	attendu := n * (n + 1) / 2
	fmt.Println("Somme calculée :", somme)
	fmt.Println("Somme attendue :", attendu)
}
