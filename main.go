package main

import (
    "strings"
    "io"
	"log"
	"net/http"
	"os"
	"fmt"
    "encoding/csv"
    "strconv"
)

type Transaction struct {
	amount int
	desc   string
}

func process_transactions(filename string) map[string]float64 {
    transaction_map := make(map[string]float64)
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
        trans_desc := strings.Split(line[1], "DES:")
        if strings.Contains(trans_desc[0], "BKOFAMERICA") {
            trans_desc[0] = "BKOFAMERICA"
        }
        trans, err := strconv.ParseFloat(line[2], 64)
        if err != nil {
            trans= 0.0
        }
        transaction_map[trans_desc[0]] += trans
    }

    return transaction_map
}

func main() {
	fs := http.FileServer(http.Dir("./ui/dist"))
	http.Handle("/", fs)
    
    trans_map := process_transactions("/Users/seanpiedmonte/stmt.csv")
    for trans, total := range trans_map {
        fmt.Println(trans, total)
    }
	log.Println("Listening on :8080...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
