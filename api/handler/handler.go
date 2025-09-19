package handler

import (
	"net/http"
	"fmt"
	"io"
	"os"
	"financesite/main/internal/config"
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
	var resp config.Response
	file, err := os.ReadFile("/Users/SeanPiedmonte/financesite/data/expenses.json")
	if err != nil {
		fmt.Println(err)
	}
	
	json.Unmarshal(file, &resp)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
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
func UploadFile(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Upload endpoint hit")

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
	tempFile, err := os.CreateTemp("", handler.Filename)
	if err != nil {
		fmt.Println(err)
	}
	defer tempFile.Close()

	// read contents into our byteArray
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		fmt.Println(err)
	}

	tempFile.Write(fileBytes)
    body := config.Response{}
    body.TransType = "expense"
    body.TransOrigin = "of"
    body.Amount = 100
    w.WriteHeader(http.StatusCreated)
    bytes, err := json.Marshal(body)
    if err != nil {
        fmt.Println(err)
    }
    w.Write(bytes)
	fmt.Fprintf(w, "Successfully Uploaded File\n")
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
