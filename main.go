package main

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/account"
	"github.com/stripe/stripe-go/v72/accountlink"
	"html/template"
	"io/ioutil"
	"lionheart/internal/user"
	"log"
	"net/http"
	"os"
	"time"
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

type Health struct {
	Status int
	Timestamp time.Time
}

func HealthHandler(w http.ResponseWriter, req *http.Request) {

	h := Health {
		Status: 200,
		Timestamp: time.Now(),
	}

	j, _ := json.Marshal(h)
	w.Header().Set("Content-Type", "application/json")
	w.Write(j)
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
	discordWebhook := "https://discord.com/api/webhooks/822043433450864650/zVJ91o_cZSJSI48yDPlW8P859qeS_L6UwdOU4KaTCw99l4Tm2Hhr-hszf7yROns0z0AX"
	data, _ := ioutil.ReadAll(r.Body)

	var event user.Event
	err := json.Unmarshal(data, &event)
	if err != nil {
		return
	}

	answers := event.Form.Answers

	name := answers[0].Text + " " + answers[1].Text
	email := answers[2].Email
	phone := answers[3].Phone
	website := answers[4].Website
	role := answers[5].Text
	description := answers[6].Text
	industry := answers[7].Choice.Label
	verticals := answers[8].Choice.Label
	legal := answers[9].Choice.Label
	help := answers[10].Text

	response :=
		"You have a new Typeform response! \n\n" +
			"Name: " + name + "\n" +
			"Email: " + email + "\n" +
			"Phone: " + phone + "\n" +
			"Website: " + website + "\n" +
			"Role: " + role + "\n" +
			"Description: " + description + "\n" +
			"Industry: " + industry + "\n" +
			"Verticals: " + verticals+ "\n" +
			"Legal: " + legal + "\n" +
			"What I want help with: " + help + "\n\n" +
			"Check typeform for more details!"

	resp := Response{
		response,
	}

	b, _ := json.Marshal(resp)

	_, err = http.Post(discordWebhook, "application/json", bytes.NewBuffer(b))
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

    webhook := "https://discord.com/api/webhooks/845954490741948416/NO9_r0MrpNLbg0v7DOgBRKXtV-eyr4XHE-HDpT90H95PVx5mSaDDha_JK2pCh_pjpvwx"

	response :=
		"You have a new 120Q response: \n\n" +
			"Name: " + u.PersonalInfo.Name + "\n" +
			"http://35.236.38.223:8888/results/" + u.PersonalInfo.Phone

	resp := Response{
		response,
	}

	b, _ := json.Marshal(resp)
	_, err = http.Post(webhook, "application/json", bytes.NewBuffer(b))

	//u.WriteUserData()
	//u.TextUser("http://35.236.38.223:8888/results/" + u.PersonalInfo.Phone)
}

func StripeHandler(w http.ResponseWriter, r *http.Request) {
	stripe.Key = os.Getenv("stripe")

	email := r.URL.Query().Get("email")

	params := &stripe.AccountParams{
		Capabilities: &stripe.AccountCapabilitiesParams{
			CardPayments: &stripe.AccountCapabilitiesCardPaymentsParams{
				Requested: stripe.Bool(true),
			},
			Transfers: &stripe.AccountCapabilitiesTransfersParams{
				Requested: stripe.Bool(true),
			},
		},
		Country: stripe.String("US"),
		Email: stripe.String(email),
		Type: stripe.String("custom"),
	}
	a, _ := account.New(params)

	params1 := &stripe.AccountLinkParams{
		Account: stripe.String(a.ID),
		RefreshURL: stripe.String("http://35.236.38.223/refresh?id=" + a.ID),
		ReturnURL: stripe.String("http://35.236.38.223/return"),
		Type: stripe.String("account_onboarding"),
	}

	al, _ := accountlink.New(params1)
	j, _ := json.Marshal(al.URL)
	w.Header().Set("Content-Type", "application/json")
	w.Write(j)
}

func RefreshHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	params1 := &stripe.AccountLinkParams{
		Account: stripe.String(id),
		RefreshURL: stripe.String("http://35.236.38.223/refresh?id=" + id),
		ReturnURL: stripe.String("http://35.236.38.223/return"),
		Type: stripe.String("account_onboarding"),
	}

	al, _ := accountlink.New(params1)
	j, _ := json.Marshal(al.URL)
	w.Header().Set("Content-Type", "application/json")
	w.Write(j)
}


func ReturnHandler(w http.ResponseWriter, r *http.Request) {
	resp := Response{
		Content: "You have a new response through Stripe onboarding!",
	}

	b, _ := json.Marshal(resp)
	http.Post("https://discord.com/api/webhooks/859000927998705704/s96-MOL0haZi2YVce5Zuam70AebeP2BgiQkq2nPL5kGDwffu1JAB1ZdkjiTzD-CWKdtz", "application/json", bytes.NewBuffer(b))
	http.Redirect(w, r, "https://www.lion.app", http.StatusSeeOther)
}


func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8888"
	}

	r := mux.NewRouter()
	r.HandleFunc("/", UsageHandler)
	r.HandleFunc("/webhook", WebhookHandler)
	r.HandleFunc("/results/{phone}", FileServer)
	r.HandleFunc("/health", HealthHandler)
	r.HandleFunc("/stripe", StripeHandler)
	r.HandleFunc("/return", ReturnHandler)
	r.HandleFunc("/refresh", RefreshHandler)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
