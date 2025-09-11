package handler

import (
	"net/http"
	"fmt"
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
func Get(w http.ResponseWriter, r *http.Request) {
	fmt.Println("GET")
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
 * description: POST Request 
 */
func Post(w http.ResponseWriter, r *http.Request) {
	fmt.Println("POST")
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
