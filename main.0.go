

package main

import (
	"fmt"
	"log"
	"os"
	"net/http"
)

func main() {

	// API routes

	// Serve files from static folder
	http.Handle("/c/", http.FileServer(http.Dir("C:\\")))
	http.Handle("/o/", http.FileServer(http.Dir("O:\\")))

	// Serve api /hi
	http.HandleFunc("/hi", func(w http.ResponseWriter, r *http.Request) {
		f, _ := os.Getwd()
		fmt.Fprintf(w, f)
	})

	port := ":5000"
	fmt.Println("Server is running on port" + port)

	// Start server on port specified above
	log.Fatal(http.ListenAndServe(port, nil))

}

