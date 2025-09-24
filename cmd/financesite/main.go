package main

import (
	"fmt"
	"net/http"
	"financesite/main/api/handler"
)

func main() {
	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir("./ui/dist"))
    
    mux.Handle("/", fs)

	mux.HandleFunc("GET /api/expenses", handler.GetExpenses)
	mux.HandleFunc("POST /api/uploadExpenses", handler.UploadExpenseFile)
	mux.HandleFunc("PUT /api/expenses", handler.Put)
	mux.HandleFunc("DELETE /api/expenses", handler.Del)

	fmt.Println("Listening on :8080...")
	
    err := http.ListenAndServe("localhost:8080", mux)
    if err != nil {
        fmt.Println(err)
    }
}
