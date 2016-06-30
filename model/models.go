package model

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

type User struct {
	gorm.Model
	Username  string
	Password  string
	Privilege string `gorm:"default:'user'"`
}

type Problem struct {
	gorm.Model
	ShortName    string
	Title        string
	Description  string `gorm:"type:text"`
	SecretBefore time.Time

	NumTestcase int
}

type Contest struct {
	gorm.Model
	Title    string
	Problems []Problem `gorm:"many2many:contest_problems"`
	StartAt  time.Time
	FinishAt time.Time
}

type ProblemInput struct {
	gorm.Model

	Username   string
	ShortName  string
	TestcaseID int

	InputFile string
}

type Submit struct {
	gorm.Model
	Username   string
	ShortName  string
	TestcaseID int

	InputFile        string
	AnswerFile       string
	SolutionFile     string
	SolutionFileName string
	Score            int
	Message          string `gorm:"type:text"`

	Status string `gorm:"default:'pending'"`
}

var DB *gorm.DB

func InitDB(host, user, pass, name string) {
	db, err := gorm.Open("postgres", fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable password=%s", host, user, name, pass))
	if err != nil {
		panic(err)
	}
	db.AutoMigrate(&User{})
	db.AutoMigrate(&Problem{})
	db.AutoMigrate(&Contest{})
	db.AutoMigrate(&ProblemInput{})
	db.AutoMigrate(&Submit{})
	db.Model(&User{}).AddUniqueIndex("idx_user_username", "username")
	db.Model(&Problem{}).AddUniqueIndex("idx_problem_shortname", "short_name")
	db.Model(&ProblemInput{}).AddUniqueIndex(
		"idx_probleminput_username_shortname_testcaseid", "username", "short_name", "testcase_id")
	DB = db
}
