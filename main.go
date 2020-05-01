package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/wcharczuk/go-chart"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
)

const cellStart = "B5"
const cellStop = "G"

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func getTeamData(team string, sheet *sheets.Service) (*team, error) {
	//Example "1fR9QhlVVlClgVXccxRV23bmEZmaJg1rup5P2nKnFXkg"
	spreadsheetId := os.Getenv("SHEET_ID")
	readRange := fmt.Sprintf("%s!%s:%s", team, cellStart, cellStop)
	resp, err := sheet.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve data from sheet: %v", err)
	}

	t := newTeam(team)

	if len(resp.Values) > 0 {
		for _, row := range resp.Values {
			date, err := time.Parse("2006-01-02", row[0].(string))
			if err != nil {
				return nil, err
			}
			cycle, err := strconv.ParseFloat(row[1].(string), 64)
			if err != nil {
				return nil, err
			}
			throughput, err := strconv.ParseFloat(row[2].(string), 64)
			if err != nil {
				return nil, err
			}
			comment := ""
			if len(row) > 5 {
				comment = row[5].(string)
			}

			data := teamData{
				date:       date,
				cycleTime:  cycle,
				throughput: throughput,
				comment:    comment,
			}
			t.data = append(t.data, data)
		}
	}
	return t, nil
}

func initSheets(credFile string) (*sheets.Service, error) {

	b, err := ioutil.ReadFile(credFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets.readonly")
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := sheets.New(client)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Sheets client: %v", err)
	}
	return srv, nil
}

func serveHTTP(res http.ResponseWriter, req *http.Request) {
	res.Write([]byte(
		"<!DOCTYPE html><html><head>" +
			"<title>team metrics</title>" +
			"<link rel=\"stylesheet\" type=\"text/css\" href=\"/main.css\">" +
			"</head>" +
			"<body>"))
	res.Write([]byte(fmt.Sprintf(`
<p><ul><li><b>cycle time</b> measures how long it takes an individual task to go through the process (lower is better)<li><b>throughput</b> measures the total amount of work delivered in a certain time period (higher is better)</ul></p>
<p>Data updated at %v</p>
`, lastUpdate)))

	l.Lock()
	for i := range charts {
		res.Write([]byte(fmt.Sprintf("<h2>%s</h2>", charts[i].title)))
		charts[i].renderer.Render(chart.SVG, res)
	}
	l.Unlock()
	res.Write([]byte("</body>"))
}

func css(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/css")
	res.Write([]byte("svg .background { fill: white; }" +
		"svg .canvas { fill: white; }"),
	)
}

type teamChart struct {
	renderer chart.Chart
	title    string
}

var l sync.Mutex
var lastUpdate time.Time
var charts []teamChart

func main() {
	credentialsFile := os.Getenv("CREDENTIALS")
	srv, err := initSheets(credentialsFile)
	if err != nil {
		log.Fatalf("Error initializing Google Sheets API: %v", err)
	}

	teams := []string{
		"DevOps",
		"VSS",
	}

	go func() {
		for {
			l.Lock()
			log.Printf("Updating chart data ...")
			charts = []teamChart{}
			for i := range teams {
				t, err := getTeamData(teams[i], srv)
				if err != nil {
					log.Printf("Error getting data for team %q: %v", teams[i], err)
					continue
				}
				charts = append(charts, teamChart{title: t.name, renderer: t.renderer()})
			}
			l.Unlock()
			log.Printf("Chart data updated")
			lastUpdate = time.Now()
			time.Sleep(time.Hour)
		}
	}()
	http.HandleFunc("/", serveHTTP)
	http.HandleFunc("/main.css", css)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
