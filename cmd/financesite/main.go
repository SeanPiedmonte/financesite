package main

import (
	"log"
	"net/http"
	"financesite/main/api/handler"
)

func main() {
	fs := http.FileServer(http.Dir("./ui/dist"))
	http.Handle("/", fs)
	
	mux := http.NewServeMux()

	mux.HandleFunc("GET /expenses", handler.GetExpenses)
	mux.HandleFunc("POST /", handler.Post)
	mux.HandleFunc("PUT /", handler.Put)
	mux.HandleFunc("DELETE /", handler.Del)

	log.Println("Listening on :8080...")
	
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
