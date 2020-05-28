package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

type emailCount struct {
	EmailAddress string
	Count        int
}

var spin *spinner.Spinner

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
	checkErr("Unable to cache oauth token", err)
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func main() {

	b, err := ioutil.ReadFile("credentials.json")
	checkErr("Unable to read client secret file", err)

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
	checkErr("Unable to parse client secret file to config", err)

	client := getClient(config)

	srv, err := gmail.New(client)
	checkErr("Unable to retrieve Gmail client", err)

	// Initialize the spinner.
	spin = spinner.New(spinner.CharSets[39], 100*time.Millisecond)

	// List all messages, then retrieve their data (2 steps).
	messageList := listAllMessages(srv)
	sortedMessageKV := getAllMessageData(srv, messageList)

	for _, m := range sortedMessageKV {
		fmt.Printf("%v: %v\n", m.EmailAddress, m.Count)
	}
}

func checkErr(msg string, err error) {
	if err != nil {
		log.Fatalf("%v: %v\n", msg, err)
	}

}

func sortSliceByIntVal(m map[string]int) []emailCount {
	var ss []emailCount
	for k, v := range m {
		ss = append(ss, emailCount{k, v})
	}

	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Count > ss[j].Count
	})

	return ss
}

func listAllMessages(srv *gmail.Service) []*gmail.Message {
	spin.Prefix = "Listing messages... "
	spin.FinalMSG = spin.Prefix + "done."
	i := 0
	messageContainer := make([]*gmail.Message, 0)
	var token string
	spin.Start()
	for {
		messageList, err := srv.Users.Messages.List("me").MaxResults(300).PageToken(token).IncludeSpamTrash(false).Do()
		checkErr("Unable to retrieve emails", err)
		thread := make([]*gmail.Message, len(messageList.Messages))
		for i, m := range messageList.Messages {
			thread[i] = m
		}
		messageContainer = append(messageContainer, thread...)
		i++
		if messageList.NextPageToken == "" {
			break
		} else {
			token = messageList.NextPageToken
		}
	}
	spin.Stop()
	fmt.Println()
	return messageContainer
}

func getAllMessageData(srv *gmail.Service, messageList []*gmail.Message) []emailCount {
	senderCount := make(map[string]int, len(messageList))
	spin.Prefix = "Retrieving all message data... "
	spin.FinalMSG = spin.Prefix + "done.\n"
	spin.Start()

	for _, msg := range messageList {
		msg, err := srv.Users.Messages.Get("me", msg.Id).Do()
		for i, header := range msg.Payload.Headers {
			if header.Name == "From" {
				// Filter out Hangouts messages
				if strings.Contains(header.Name, "profiles.google.com") {
					break
				}
				// Add email sender to our tally
				_, exists := senderCount[msg.Payload.Headers[i].Value]
				if exists {
					senderCount[msg.Payload.Headers[i].Value]++
				} else {
					senderCount[msg.Payload.Headers[i].Value] = 1
				}
				// Stop iterating over headers
				break
			}
		}
		checkErr("Trouble recieving message", err)

	}
	spin.Stop()
	fmt.Println()

	return sortSliceByIntVal(senderCount)
}
