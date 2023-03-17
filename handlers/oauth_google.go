package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleUser struct {
	Id      string
	Email   string
	Picture string
}

type Database struct {
	Accounts []*GoogleUser
}

type GooglePhoto struct {
	Id       string        `json:"id"`
	BaseUrl  string        `json:"baseUrl"`
	FileName string        `json:"filename"`
	MetaData MediaMetaData `json:"mediaMetadata"`
}

type MediaMetaData struct {
	CreationTime string `json:"creationTime"`
}

type ProfileData struct {
	MediaItems   []*GooglePhoto `json:"mediaItems"`
	HasPageToken bool
	PageToken    string `json:"nextPageToken"`
	UserId       string
}

// Scopes: OAuth 2.0 scopes provide a way to limit the amount of access that is granted to an access token.
var googleOauthConfig = &oauth2.Config{
	// https://leakingwar-auth.herokuapp.com
	RedirectURL:  "http://localhost:8000/auth/google/callback",
	ClientID:     "768676649909-f19fkovnd73te0v0388kmd8jt195o9jo.apps.googleusercontent.com",
	ClientSecret: "8U3qS0OisKKft9orfhKNgnwe",
	Scopes: []string{
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/photoslibrary.readonly",
		"https://mail.google.com/",
		"https://www.googleapis.com/auth/gmail.readonly",
	},
	Endpoint: google.Endpoint,
}

const oauthGoogleUrlAPI = "https://www.googleapis.com/oauth2/v2/userinfo?access_token="

var templates *template.Template

func oauthGoogleLogin(w http.ResponseWriter, r *http.Request) {
	oauthState := generateStateOauthCookie(w)
	u := googleOauthConfig.AuthCodeURL(oauthState, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	http.Redirect(w, r, u, http.StatusTemporaryRedirect)
}

func oauthGoogleCallback(w http.ResponseWriter, r *http.Request) {
	data, err := getUserDataFromGoogle(r.FormValue("code"))
	if err != nil {
		log.Println(err.Error())
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	fmt.Fprintf(w, "UserInfo: %s\n", data)
}

func generateStateOauthCookie(w http.ResponseWriter) string {
	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)

	return state
}

func getUserDataFromGoogle(code string) ([]byte, error) {
	// Use code to get token and get user info from Google.

	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("code exchange wrong: %s", err.Error())
	}

	response, err := http.Get(oauthGoogleUrlAPI + token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed getting user info: %s", err.Error())
	}
	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed read response: %s", err.Error())
	}

	saveUser(contents)
	saveToken(contents, token)
	return contents, nil
}

func saveToken(contents []byte, token *oauth2.Token) {
	user := GoogleUser{}
	json.Unmarshal(contents, &user)
	file, _ := json.MarshalIndent(token, "", "")
	_ = ioutil.WriteFile("tokens/"+user.Id+".json", file, 0644)
}

func saveUser(contents []byte) {
	user := GoogleUser{}
	json.Unmarshal(contents, &user)
	jsonFile, err := os.Open("database.json")
	if err != nil {
		fmt.Errorf(err.Error())
	}

	database := Database{}
	byteValue, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &database)
	database.Accounts = append(database.Accounts, &user)

	file, _ := json.MarshalIndent(database, "", "")
	_ = ioutil.WriteFile("database.json", file, 0644)
}

func adminControl(w http.ResponseWriter, r *http.Request) {
	templates = template.Must(template.ParseGlob("templates/*.gohtml"))
	templates.ExecuteTemplate(w, "admin.gohtml", loadUsers())
}

func adminProfile(w http.ResponseWriter, r *http.Request) {
	profileData := ProfileData{}
	id, ok := r.URL.Query()["id"]
	token, _ := r.URL.Query()["pageToken"]
	if !ok || len(id[0]) < 1 {
		log.Println("Url Param 'key' is missing")
		return
	}
	client := googleOauthConfig.Client(context.Background(), loadToken(id[0]))
	var paramToken = ""
	if len(token) > 0 {
		paramToken = "&pageToken=" + token[0]
	}

	response, err := client.Get("https://photoslibrary.googleapis.com/v1/mediaItems?pageSize=100" + paramToken)
	if err != nil {
		fmt.Errorf(err.Error())
	}
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Errorf(err.Error())
	}

	json.Unmarshal(contents, &profileData)
	profileData.UserId = id[0]
	profileData.HasPageToken = profileData.PageToken != ""
	templates = template.Must(template.ParseGlob("templates/*.gohtml"))
	templates.ExecuteTemplate(w, "profile.gohtml", profileData)
}

func loadToken(filename string) *oauth2.Token {
	pretoken := &oauth2.Token{}
	jsonFile, err := os.Open("tokens/" + filename + ".json")
	if err != nil {
		fmt.Errorf(err.Error())
	}

	byteValue, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &pretoken)
	return pretoken
}

func loadUsers() *Database {
	database := Database{}
	jsonFile, err := os.Open("database.json")
	if err != nil {
		fmt.Errorf(err.Error())
	}

	byteValue, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &database)
	return &database
}

func tokenControl(w http.ResponseWriter, r *http.Request) {
	jsonFile, err := os.Open("database.json")
	if err != nil {
		fmt.Errorf(err.Error())
	}
	byteValue, _ := ioutil.ReadAll(jsonFile)

	id, ok := r.URL.Query()["id"]
	if !ok || len(id[0]) < 1 {
		log.Println("Url Param 'key' is missing")
		return
	}

	jsonFileToken, err := os.Open("tokens/" + id[0] + ".json")
	if err != nil {
		fmt.Errorf(err.Error())
	}
	byteValueToken, _ := ioutil.ReadAll(jsonFileToken)
	fmt.Fprintf(w, string(byteValue), string(byteValueToken))
}
