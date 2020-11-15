package db

import (
	"errors"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Level struct {
	gorm.Model
	EventID               int
	SponsorID             int
	Name                  string `gorm:"column:level_name"`
	Cost                  string
	MaxNumberOfSponsors   int
	MaxNumberOfFreeBadges int
}

type Member struct {
	gorm.Model
	Name      string
	Email     string
	SponsorID int
}

type Sponsor struct {
	gorm.Model
	ID      int
	EventID int
	Name    string
	Level   Level `gorm:"foreignKey:Name"`
	Members []Member
}

type Event struct {
	gorm.Model
	ID       int
	Name     string `gorm:"column:event_name"`
	Levels   []Level
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
		Name:    name,
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

func CreateMember(m struct {
	Name  string
	Email string
}) {
	Database.Create(&Member{
		Name:  m.Name,
		Email: m.Email,
	})
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
