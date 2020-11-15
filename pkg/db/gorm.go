package db

import (
	"errors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Level struct {
	gorm.Model
	EventID int
	SponsorID int
	Name string `gorm:"column:level_name"`
	Cost string
	MaxNumberOfSponsors int
	MaxNumberOfFreeBadges int
}

type Member struct {
	gorm.Model
	Name string
	Email string
	SponsorID int
}

type Sponsor struct {
	gorm.Model
	ID int
	EventID int
	Name string
	Level Level `gorm:"foreignKey:Name"`
	Members []Member
}

type Event struct {
	gorm.Model
	ID int
	Name string `gorm:"column:event_name"`
	Levels []Level
	Sponsors []Sponsor
}

func initialMigration(db *gorm.DB) {
	db.AutoMigrate(&Level{})
	db.AutoMigrate(&Member{})
	db.AutoMigrate(&Sponsor{})
	db.AutoMigrate(&Event{})
}

func CreateSponsor(name string, eventId int) *Sponsor {
	sponsor := Sponsor{
		Name: name,
		EventID: eventId,
	}
	Database.Create(&sponsor)

	return &sponsor
}

func GetEvent(id int) (*Event, error) {
	var event Event
	var error error
	err := Database.First(&event, id)
	if errors.Is(err.Error, gorm.ErrRecordNotFound) {
		error = gorm.ErrRecordNotFound
	}
	return &event, error
}

func CreateEvent(name string) *Event {
	var event Event
	event.Name = name
	Database.Create(&event)

	return &event
}

func CreateMember(m struct{
	Name string
	Email string
}) {
	Database.Create(&Member{
		Name: m.Name,
		Email: m.Email,
	})
}

// Initialize variables
var Database *gorm.DB

func InitDB() {
	var err error
	Database, err = gorm.Open(postgres.New(postgres.Config{
		DSN: "host=localhost user=user password=hey dbname=postgres port=5432 sslmode=disable",
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	initialMigration(Database)
}

