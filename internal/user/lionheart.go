package user

import (
	"bufio"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/db"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
)

var fb *db.Client
var data map[int]Question

func init() {
	if fb == nil {
		opt := option.WithCredentialsJSON([]byte(os.Getenv("ACCOUNT")))
		config := &firebase.Config{
			DatabaseURL: os.Getenv("DB_URL"),
		}

		f, _ := firebase.NewApp(context.Background(), config, opt)
		fb, _ = f.Database(context.Background())
	}
	data = map[int]Question{}
}

type Personal struct {
	Name string
	Email string
	Phone string
	MorF  string
	Gender string
	Ethnicity string
	Education string
	Country string
	USA     bool
	State	string
	City    string
	Live    string
	Religion string
	Marital string
}

type User struct {
	PersonalInfo Personal
	Subtraits    map[string]*Trait
	Traits       map[string]*Trait
}

type Trait struct {
	Name        string
	RawScore    float64
	NormalScore float64
	Min         float64
}

type Question struct {
	Number      int
	Description string
	Key         int
	Trait       string
	Min         float64
}

type Event struct {
	Id   string `json:"event_id"`
	Type string `json:"event_type"`
	Form Form   `json:"form_response"`
}

type Form struct {
	Id         string     `json:"form_id"`
	Token      string     `json:"token"`
	Landed     string     `json:"landed_at"`
	Submitted  string     `json:"submitted_at"`
	Definition Definition `json:"definition"`
	Answers    []Answer   `json:"answers"`
}

type Definition struct {
	Id     string          `json:"id"`
	Title  string          `json:"title"`
	Fields []QuestionField `json:"fields"`
}

type QuestionField struct {
	Id         string      `json:"id"`
	Title      string      `json:"title"`
	Type       string      `json:"type"`
	Ref        string      `json:"ref"`
	Properties interface{} `json:"properties"`
}

type Answer struct {
	Type    string      `json:"type"`
	Boolean bool        `json:"boolean"`
	Text    string      `json:"text"`
	Email   string      `json:"email"`
	Phone   string      `json:"phone_number"`
	Number  int         `json:"number"`
	Choice  Choice      `json:"choice"`
	Field   AnswerField `json:"field"`
}

type Choice struct {
	Label  string `json:"label"`
}

type AnswerField struct {
	Id   string `json:"id"`
	Type string `json:"type"`
	Ref  string `json:"ref"`
}

func (u *User) LoadQuestionsFromFile() {
	file, err := os.Open(os.Getenv("TESTFILE"))
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()

		question := strings.Split(text, "|")
		number, _ := strconv.Atoi(question[0])
		key, _ := strconv.Atoi(question[2])
		min, _ := strconv.Atoi(question[4])

		q := Question{
			Number:      number,
			Description: question[1],
			Key:         key,
			Trait:       question[3],
			Min:         float64(min),
		}

		data[number] = q
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func (u *User) ProcessUserInfo(test []byte) {
	var event Event

	err := json.Unmarshal(test, &event)
	if err != nil {
		return
	}

	answers := event.Form.Answers

	u.PersonalInfo.Name = answers[0].Text
	u.PersonalInfo.Email = answers[1].Email
	u.PersonalInfo.Phone = answers[2].Phone
	u.PersonalInfo.MorF = answers[3].Choice.Label
	u.PersonalInfo.Gender = answers[4].Choice.Label
	u.PersonalInfo.Ethnicity = answers[5].Choice.Label
	u.PersonalInfo.Education = answers[6].Choice.Label
	u.PersonalInfo.Country = answers[7].Choice.Label
	u.PersonalInfo.USA = answers[8].Boolean
	u.PersonalInfo.State = answers[9].Choice.Label
	u.PersonalInfo.City = answers[10].Choice.Label
	u.PersonalInfo.Live = answers[11].Choice.Label
	u.PersonalInfo.Religion = answers[12].Choice.Label
	u.PersonalInfo.Marital = answers[13].Choice.Label
}

func (u *User) ProcessSubtraits(test []byte) {
	var event Event
	subs := make(map[string]*Trait)

	err := json.Unmarshal(test, &event)
	if err != nil {
		return
	}

	for i, ele := range event.Form.Answers[14:] {

		entry := data[i+1]

		if entry.Trait == "" {
			continue
		}

		if _, ok := subs[entry.Trait]; ok {
			subs[entry.Trait].RawScore += float64(entry.Key * ele.Number)
		} else {
			subs[entry.Trait] = &Trait{
				entry.Trait,
				float64(entry.Key * ele.Number),
				0,
				entry.Min,
			}
		}
	}

	u.Subtraits = subs
}

func (u *User) NormalizeSubtraits() {
	for _, v := range u.Subtraits {
		s := 6.25 * (v.RawScore - v.Min)
		m := math.RoundToEven(s)
		v.NormalScore = m
	}
}

func (u *User) ProcessTraits() {
	traits := make(map[string]*Trait)

	for k, v := range u.Subtraits {

		letter := string(k[0])

		if _, ok := traits[letter]; ok {
			traits[letter].RawScore += v.RawScore
		} else {
			traits[letter] = &Trait{
				letter,
				v.RawScore,
				0,
				getMin(letter),
			}
		}
	}

	u.Traits = traits
}

func (u *User) NormalizeTraits() {
	for _, v := range u.Traits {
		s := (100 / float64(96)) * (v.RawScore - v.Min)
		m := math.RoundToEven(s)
		v.NormalScore = m
	}
}

func (u *User) TextUser(link string) {
	accountSid := os.Getenv("ACCOUNT_SID")
	token := os.Getenv("TOKEN")
	urlStr := "https://api.twilio.com/2010-04-01/Accounts/" + accountSid + "/Messages.json"

	msgData := url.Values{}
	msgData.Set("To", u.PersonalInfo.Phone)
	msgData.Set("From", os.Getenv("PHONE"))
	msgData.Set("Body", "Hello! Your Lionheart test results can be found at: " + link)
	msgDataReader := *strings.NewReader(msgData.Encode())

	client := &http.Client{}
	req, err := http.NewRequest("POST", urlStr, &msgDataReader)
	if err != nil {
		panic(err)
	}
	req.SetBasicAuth(accountSid, token)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	client.Do(req)
}

func (u *User) WriteUserData() {

	err := fb.NewRef("/users/" + u.PersonalInfo.Phone).Set(context.Background(), u)
	if err != nil {
		return
	}
}

func getMin(letter string) float64 {

	switch letter {
	case "A":
		return -66
	case "C":
		return -36
	case "E":
		return 6
	case "N":
		return -66
	case "O":
		return -78
	}

	return 0
}