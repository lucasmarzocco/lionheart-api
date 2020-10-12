package main

import (
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"

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

func FileServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	phone := vars["phone"]
	http.ServeFile(w, r, "internal/reports/"+phone+"_report")
}

func UsageHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("server is running"))
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
	r.HandleFunc("/results/{phone}", FileServer)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
