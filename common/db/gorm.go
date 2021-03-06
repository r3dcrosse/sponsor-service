package db

import (
	"errors"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Level struct {
	gorm.Model
	ID                    int `gorm:"primary_key"`
	EventID               int
	Name                  string
	Cost                  string
	MaxNumberOfSponsors   int
	MaxNumberOfFreeBadges int
}

type Member struct {
	gorm.Model
	ID        int `gorm:"primary_key"`
	Name      string
	Email     string
	SponsorID int
	EventID   int
}

type Sponsor struct {
	gorm.Model
	ID        int `gorm:"primary_key"`
	EventID   int
	Name      string
	LevelID   int
	LevelName string
	Level     Level
	Members   []Member
}

type Event struct {
	gorm.Model
	ID             int `gorm:"primary_key"`
	EventServiceID int
	Name           string
	Levels         []Level
	Sponsors       []Sponsor
}

func initialMigration(db *gorm.DB) {
	db.AutoMigrate(&Level{})
	db.AutoMigrate(&Member{})
	db.AutoMigrate(&Sponsor{})
	db.AutoMigrate(&Event{})
}

func CreateMember(name string, email string, sponsorId int) *Member {
	member := Member{
		Name:      name,
		Email:     email,
		SponsorID: sponsorId,
	}
	Database.Create(&member)

	return &member
}

func GetLevel(id int) (*Level, error) {
	var level Level
	var error error
	err := Database.First(&level, id)
	if errors.Is(err.Error, gorm.ErrRecordNotFound) {
		error = gorm.ErrRecordNotFound
	}
	return &level, error
}

func CreateLevel(name string, cost string, maxNumSponsors int, maxNumBadges int, eventId int) *Level {
	level := Level{
		Name:                  name,
		EventID:               eventId,
		MaxNumberOfSponsors:   maxNumSponsors,
		MaxNumberOfFreeBadges: maxNumBadges,
		Cost:                  cost,
	}
	Database.Create(&level)

	event := Event{}
	// Get Levels array from Event and update it
	Database.First(&event, eventId)
	event.Levels = append(event.Levels, level)
	Database.Save(&event)

	return &level
}

func UpdateLevel(id int, name string, cost string, maxNumSponsors int, maxNumBadges int, eventId int) (*Level, error) {
	var level Level
	err := Database.First(&level, id)
	level.Name = name
	level.Cost = cost
	level.MaxNumberOfSponsors = maxNumSponsors
	level.MaxNumberOfFreeBadges = maxNumBadges
	level.EventID = eventId
	Database.Save(&level)

	return &level, err.Error
}

func CreateSponsorWithLevel(name string, levelId int, eventId int) *Sponsor {
	level := Level{}
	Database.First(&level, levelId)

	sponsor := Sponsor{
		Name:      name,
		EventID:   eventId,
		LevelID:   levelId,
		LevelName: level.Name,
		Level:     level,
	}
	Database.Create(&sponsor)

	event := Event{}
	Database.First(&event, eventId)
	event.Sponsors = append(event.Sponsors, sponsor)
	Database.Save(&event)

	return &sponsor
}

func CreateSponsor(name string, eventId int) *Sponsor {
	sponsor := Sponsor{
		Name:    name,
		EventID: eventId,
	}
	Database.Create(&sponsor)

	event := Event{}
	// Get Sponsors array from event and update it
	Database.First(&event, eventId)
	event.Sponsors = append(event.Sponsors, sponsor)
	Database.Save(&event)

	return &sponsor
}

func GetSponsor(id int) (*Sponsor, error) {
	var sponsor Sponsor
	var error error
	err := Database.First(&sponsor, id)
	if errors.Is(err.Error, gorm.ErrRecordNotFound) {
		error = gorm.ErrRecordNotFound
	}
	return &sponsor, error
}

func GetEvent(id int, eventServiceId int) (*Event, error) {
	var levels []Level
	var sponsors []Sponsor
	var event Event
	var error error

	idToUse := id
	if idToUse == -1 {
		idToUse = eventServiceId
		err := Database.Where(&Event{EventServiceID: idToUse}).First(&event)
		if errors.Is(err.Error, gorm.ErrRecordNotFound) {
			error = gorm.ErrRecordNotFound
		}
	} else {
		idToUse = id
		err := Database.First(&event, idToUse)
		if errors.Is(err.Error, gorm.ErrRecordNotFound) {
			error = gorm.ErrRecordNotFound
		}
	}

	Database.Where(&Level{EventID: id}).Find(&levels)
	Database.Where(&Sponsor{EventID: id}).Find(&sponsors)
	event.Sponsors = sponsors
	event.Levels = levels

	return &event, error
}

func UpdateEvent(eventId int, eventName string) (*Event, error) {
	var event Event
	var levels []Level
	var error error
	err := Database.First(&event, eventId)
	if errors.Is(err.Error, gorm.ErrRecordNotFound) {
		error = gorm.ErrRecordNotFound
	}

	Database.Where(&Level{EventID: eventId}).Find(&levels)
	if levels != nil {
		event.Levels = levels
	}

	if err != nil {
		event.Name = eventName
		Database.Save(&event)
	}

	return &event, error
}

func GetAllEvents() *[]Event {
	var events []Event
	Database.Find(&events)

	for i, _ := range events {
		var levels []Level
		var sponsors []Sponsor

		Database.Where(&Level{EventID: events[i].ID}).Find(&levels)
		Database.Where(&Sponsor{EventID: events[i].ID}).Find(&sponsors)

		events[i].Sponsors = sponsors
		events[i].Levels = levels
	}

	return &events
}

func CreateEvent(name string, eventId int) *Event {
	var event Event
	event.Name = name
	event.EventServiceID = eventId
	Database.Create(&event)

	return &event
}

// Initialize variables
var Database *gorm.DB

type Creds struct {
	Host     string
	Port     string
	User     string
	Password string
	Dbname   string
	Sslmode  string
}

func InitDB(o Creds) {
	var err error
	Database, err = gorm.Open(postgres.New(postgres.Config{
		DSN:                  fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", o.Host, o.Port, o.User, o.Password, o.Dbname, o.Sslmode),
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	initialMigration(Database)
}
