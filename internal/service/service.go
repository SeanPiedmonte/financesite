package service

import (
	"bytes"
    "strings"
    "io"
	"log"
	"os"
	"fmt"
    "encoding/csv"
    "strconv"
)


/*
 * name: closeFile
 * 
 * parameters
 *  f *os.File: file to be closed
 *
 * return:
 *
 * description: Close a File
 */
func CloseFile(f *os.File) {
	err := f.Close()

	if err != nil {
		log.Fatal(err)
    }
}

/*
 * name: get_balance
 * 
 * parameters:
 * 	account_name string: name of the account we want to attach the balance to	
 *  filename string: name of the file to be processed
 *  
 * return: float64 
 *
 * description: Get the balance from the account file being submitted.
 * 			    Used to calculate net worth. Returns a float of the balance
 *
 */
func GetBalance(filename string) float64 {
	file, err := os.OpenFile(filename, os.O_RDWR, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer CloseFile(file)

	csv_reader := csv.NewReader(file)
	line, err := csv_reader.Read()
	if err != nil {
		log.Fatal(err)
	}
	balance_pos := -1
	for i, elem := range line {
		elem = strings.ToLower(elem)
		if strings.Contains(elem, "balance") {
			balance_pos = i
		} else if strings.Contains(elem, "running bal") {
			data, err := csv_reader.ReadAll()
			if err != nil {
				log.Fatal(err)
			}
			balance, err := strconv.ParseFloat(strings.ReplaceAll(data[len(data)-1][len(data[0])-1], ",", ""), 64) 
			return balance
		}
	}
	if balance_pos == -1 {
		log.Fatal("No Balance Tag Found")
		return 0
	}
	line, err = csv_reader.Read()
	if err != nil {
		log.Fatal()
	}

	line[balance_pos] = strings.ReplaceAll(line[balance_pos], ",", "")
	balance, err := strconv.ParseFloat(line[balance_pos], 64)
	if err != nil {
		log.Fatal(err)
	}
	return balance 
}

/*
 * name: process_transactions
 *
 * parameters:
 * 	filename string: file being processed
 *
 * return: map[string]float64
 * 
 * description: Processes the transaction on either a debit or credit card
 *				account to see where money is being sent or spent. A map
 *				is returned of transactions and the total positive or negative
 *				value.
 */
func ProcessTransactions(fileBytes []byte) map[string]float64 {
    transaction_map := make(map[string]float64)
	
	bytesReader := bytes.NewReader(fileBytes) 
    csv_reader := csv.NewReader(bytesReader)
	

	i := 0
    for {
        line, err := csv_reader.Read()
        if err == io.EOF {
            break
        } else if err != nil {
            fmt.Println("Error reading line:", err)
			break
        }

		if i == 0 || i == 1{
			i += 1
			continue
		}
		
		line[2] = strings.ReplaceAll(line[2], ",","")
        trans, err := strconv.ParseFloat(line[2], 64)
        if err != nil {
            trans= 0.0
        }

		// Have to take the substring at 10 due to dates and transaction ids
		// so taking the substring reduces the size of our map and easier to
		// map transactions to their origin
		vendor := line[1][0:11]
        transaction_map[vendor] += trans
    }

    return transaction_map
}

