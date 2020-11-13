package main

//////////////////////////////////////////////////////////////
//
// Anti-Corruption Layer Models
//
//////////////////////////////////////////////////////////////
// Sponsor struct
type Sponsor struct {
	Event string `json:"event"`
	Level Level `json:"level"`
	Representatives []Representative `json:"representatives"`
}

// Level struct
type Level struct {
	Name string `json:"name"`
	Cost string `json:"cost"`
	NumberOfBadges int `json:"number_of_badges"`
}

// Representative struct
type Representative struct {
	Name string `json:"name"`
	Email string `json:"email"`
}

//////////////////////////////////////////////////////////////
//
// Our Microservice Models
//
//////////////////////////////////////////////////////////////
// Event struct
type Event struct {
	Name string
	Levels []Level
	Sponsors []Sponsor
}

