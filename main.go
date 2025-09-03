package main

import (
	"log"
	"net/http"
	"bufio"
	"os"
	"fmt"
)

type Transaction struct {
	amount int
	desc   string
}

func process_transactions(filename string) {
	file, err := os.OpenFile(filename, os.O_RDWR, 0644)
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(file)
	fmt.Println("%v", scanner)
}

func main() {
	fs := http.FileServer(http.Dir("./ui/dist"))
	http.Handle("/", fs)

	log.Println("Listening on :8080...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
