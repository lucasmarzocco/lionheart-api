package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"fmt"

	"lionheart/internal/user"

	"github.com/gorilla/mux"
)

type Results struct {
	Name      string
	Traits    map[string]*user.Trait
	Subtraits map[string]*user.Trait
	A         float64
	C         float64
	E         float64
	O         float64
	N         float64
}

type Response struct {
	Content  string `json:"content"`
}

func FileServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	phone := vars["phone"]
	http.ServeFile(w, r, "internal/reports/"+phone+"_report")
}

func UsageHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("server is running"))
}

func ApprenticeHandler(w http.ResponseWriter, r *http.Request) {
	discordWebhook := "https://discord.com/api/webhooks/818693274385907773/LeVc2StaL_sOyKYf7bCbSniYWn8dk-bEx5U40v6Er3RPRgPkkA1MrYQ_pK96QwyhltaN"
	data, _ := ioutil.ReadAll(r.Body)

	var event user.Event
	err := json.Unmarshal(data, &event)
	if err != nil {
		return
	}

	answers := event.Form.Answers

	name := answers[0].Text + " " + answers[1].Text
	email := answers[2].Email
	//dob := answers[3].Date
	located := answers[4].Choice.Label
	earliest := answers[5].Date
	//daysAvailable := answers[6].Choices.Labels
	//hoursAvailable := answers[7].Choice.Label
	//areas := answers[8].Choices.Labels
	//industries := answers[9].Choice.Label

	response :=
		"You have a new Typeform response! \n\n" +
		"Name: " + name + "\n" +
		"Email: " + email + "\n" +
		"Location: " + located + "\n" +
		"Earliest Start Date: " + earliest + "\n\n" +
		"Check typeform for more details!"

	resp := Response{
		response,
	}

	b, _ := json.Marshal(resp)

	_, err = http.Post(discordWebhook, "application/json", bytes.NewBuffer(b))
}

func BusinessHandler(w http.ResponseWriter, r *http.Request) {
	data, _ := ioutil.ReadAll(r.Body)

	fmt.Println(string(data))
}


func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	data, _ := ioutil.ReadAll(r.Body)

	u := &user.User{}
	u.LoadQuestionsFromFile()
	u.ProcessSubtraits(data)
	u.ProcessUserInfo(data)
	u.NormalizeSubtraits()
	u.ProcessTraits()
	u.NormalizeTraits()

	x := Results{
		u.PersonalInfo.Name,
		u.Traits,
		u.Subtraits,
		u.Traits["A"].NormalScore,
		u.Traits["C"].NormalScore,
		u.Traits["E"].NormalScore,
		u.Traits["O"].NormalScore,
		u.Traits["N"].NormalScore,
	}

	if string(u.PersonalInfo.Phone[0]) == "+" {
		u.PersonalInfo.Phone = u.PersonalInfo.Phone[1:]
	}

	f, err := os.Create("internal/reports/" + u.PersonalInfo.Phone + "_report")
	if err != nil {
		panic(err)
	}

	t, err := template.ParseFiles("internal/resources/chart.html")
	if err != nil {
		panic(err)
	}

	err = t.Execute(f, x)
	if err != nil {
		panic(err)
	}

	u.WriteUserData()
	u.TextUser(" https://lionheart-api.herokuapp.com/results/" + u.PersonalInfo.Phone)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	r := mux.NewRouter()
	r.HandleFunc("/", UsageHandler)
	r.HandleFunc("/webhook", WebhookHandler)
	r.HandleFunc("/webhook/apprentice", ApprenticeHandler)
	r.HandleFunc("/webhook/business", BusinessHandler)
	r.HandleFunc("/results/{phone}", FileServer)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
