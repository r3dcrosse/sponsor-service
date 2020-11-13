package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// Initialize data
var events []Event
var sponsors []Sponsor

// Get a list of sponsor organization names and each sponsor's level for an event
func getSponsorsForEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r) // Gets params

	// Looping through events to find the one from our request
	for _, event := range events {
		if event.Name == params["event"] {
			json.NewEncoder(w).Encode(event)
			return
		}
	}

	// Return an empty event if none is found
	json.NewEncoder(w).Encode(&Event{
		Name:     "",
		Levels:   nil,
		Sponsors: nil,
	})
}

// To create a sponsor
func createSponsor(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var sponsor Sponsor
	err := json.NewDecoder(r.Body).Decode(&sponsor)
	if err != nil {
		fmt.Println(err)
	}

	sponsors = append(sponsors, sponsor)
	json.NewEncoder(w).Encode(sponsor)
}



func main() {
	// Initialize the router
	router := mux.NewRouter()

	// Hardcoded data - @todo: add database

	// Route handles and endpoints
	router.HandleFunc("/sponsor/{event}", getSponsorsForEvent).Methods("GET") // show a list of sponsor organization names and each sponsor's level for an event
	router.HandleFunc("/sponsor", createSponsor).Methods("POST") // create a sponsor at a specific level
	//router.HandleFunc("/sponsor/{id}", updateSponsor).Methods("PUT") // add people on the sponsors team
	//router.HandleFunc("/sponsor/{id}", removeSponsor).Methods("DELETE") // remove people on the sponsors team

	// Start server
	log.Fatal(http.ListenAndServe(":8000", router))
}