package main

import (
    "io"
	"log"
	"net/http"
	"os"
	"fmt"
    "encoding/csv"
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
    csv_reader := csv.NewReader(file)
    for {
        line, err := csv_reader.Read()
        if err == io.EOF {
            break
        }
        if err != nil {
            fmt.Println("Error reading line:", err)
        }
        fmt.Println(line)
    }
}

func main() {
	fs := http.FileServer(http.Dir("./ui/dist"))
	http.Handle("/", fs)
    
    process_transactions("/Users/seanpiedmonte/stmt.csv")
	log.Println("Listening on :8080...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
