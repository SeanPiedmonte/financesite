package handler

import (
	"net/http"
	"fmt"
	"io"
	"os"
	"financesite/main/internal/config"
	"financesite/main/internal/service"
	"encoding/json"
)


/*
 * name: get 
 * 
 * parameters
 *  w http.ResponseWriter: How we write back responses
 *  r *http.Request: The request we are getting from the client
 *
 * return:
 *
 * description: GET Request 
 */
func GetExpenses(w http.ResponseWriter, r *http.Request) {
	fmt.Println("GET REQUEST")
	fileData, err := os.ReadFile("/home/seanpiedmonte/financesite/data/expenses.json")
	if err != nil {
		fmt.Println(err)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fileData)
}

/*
 * name: post 
 * 
 * parameters
 *  w http.ResponseWriter: How we write back responses
 *  r *http.Request: The request we are getting from the client
 *
 * return:
 *
 * description: Handles our file uploads from the client to then be processed
 * 				and sent back to the client.
 */ 
func UploadExpenseFile(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Upload endpoint hit")
	
	fmt.Println(r.Header)
	// Parse a max of 10 MB files.
	r.ParseMultipartForm(10 << 20)

	// Returns the first file recieved under the key 'FILEKEY'
	// and the file header so we get the filename, HEADER, and filesize
	file, handler, err := r.FormFile("FILEKEY")
	if err != nil {
		fmt.Println("Error Retrieving the File")
		fmt.Println(err)
		return
	}
	defer file.Close()
	fmt.Printf("Uploaded file: %+v\n", handler.Filename)
	fmt.Printf("File Size: %+v\n", handler.Size)
	fmt.Printf("MIME Header: %+v\n", handler.Header)

	// Create our Temporary file
	expenseFile, err := os.Create("/home/seanpiedmonte/financesite/data/expenses.json")
	if err != nil {
		fmt.Println(err)
	}
	defer expenseFile.Close()

	// read contents into our byteArray
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		fmt.Println(err)
	}

	//tempFile.Write(fileBytes)
	transactions, err := service.ProcessTransactions(fileBytes)
	if err != nil {
		fmt.Println(err)
	}
	
	
	var response []config.Response
	for k, v := range transactions {
		if v < 0.0 {
			response = append(response, config.Response{"Expense", k, v})
		}
	}


	w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
	json.NewEncoder(expenseFile).Encode(response)
    /*data, err := json.Marshal(response)
    if err != nil {
        fmt.Println(err)
    }

	wBytes, err := expenseFile.Write(data)
	if err != nil {
		fmt.Println(wBytes)
		fmt.Println(err)
	}*/
}

/*
 * name: put 
 * 
 * parameters
 *  w http.ResponseWriter: How we write back responses
 *  r *http.Request: The request we are getting from the client
 *
 * return:
 *
 * description: PUT Request 
 */
func Put(w http.ResponseWriter, r *http.Request) {
	fmt.Println("PUT")
}

/*
 * name: del 
 * 
 * parameters
 *  w http.ResponseWriter: How we write back responses
 *  r *http.Request: The request we are getting from the client
 *
 * return:
 *
 * description: DELETE Request 
 */
func Del(w http.ResponseWriter, r *http.Request) {
	fmt.Println("DEL")
}
