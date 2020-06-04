package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/schollz/progressbar"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

type emailCount struct {
	EmailAddress string
	Count        int
}

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

func readCliArgs() (int, int, string) {
	offset := flag.Int("o", 0, "number of messages to offset")
	limit := flag.Int("l", 0, "number of messages to retrieve")
	creds := flag.String("c", "credentials.json", "location of credentials.json")
	flag.Parse()
	return *offset, *limit, *creds
}

func main() {
	offset, limit, creds := readCliArgs()

	b, err := ioutil.ReadFile(creds)
	checkErr("Unable to read client secret file", err)

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
	checkErr("Unable to parse client secret file to config", err)

	client := getClient(config)

	srv, err := gmail.New(client)
	checkErr("Unable to retrieve Gmail client", err)

	// List all messages, then retrieve their data (2 steps).
	messageList := listAllMessages(srv, offset, limit)
	getAllMessageData(srv, &messageList)

	for _, m := range messageList {
		if m.Payload != nil {
			for _, header := range m.Payload.Headers {
				if header.Name == "From" {
					fmt.Printf("%+v\n", header.Value)
					break
				}
			}
		}
	}
}

func checkErr(msg string, err error) {
	if err != nil {
		log.Fatalf("%v: %v\n", msg, err)
	}

}

func listAllMessages(srv *gmail.Service, offset int, limit int) []*gmail.Message {
	i := 0
	pageLength := 500
	messageContainer := make([]*gmail.Message, 0)
	bar := progressbar.Default(-1, "Retrieving list of messages...")
	var token string
	log.Printf("Paging size for message retrieval: %v\n", pageLength)
	for {
		// Retrieve list of messages
		messageList, err := srv.Users.Messages.List("me").MaxResults(int64(pageLength)).PageToken(token).IncludeSpamTrash(false).Do()
		checkErr("Unable to retrieve emails", err)

		// Unravel the thread into a slice
		thread := make([]*gmail.Message, len(messageList.Messages))
		for i, m := range messageList.Messages {
			thread[i] = m
		}

		// Add them to the external container
		messageContainer = append(messageContainer, thread...)

		i++
		bar.Add(1)
		if messageList.NextPageToken == "" {
			break
		} else {
			token = messageList.NextPageToken
		}
	}
	bar.Finish()
	fmt.Println()

	// If the limit is defined, just do the offset
	if limit == 0 {
		return messageContainer[offset:]
	}
	return messageContainer[offset : offset+limit]
}

func getAllMessageData(srv *gmail.Service, messageList *[]*gmail.Message) {
	// Gmail ListMessage API doesn't return any info other than ID - this pulls all other info
	fmt.Println("Retrieving all message data...")
	bar := progressbar.Default(int64(len(*messageList)))
	for _, msg := range *messageList {
		fullMsg, err := srv.Users.Messages.Get("me", msg.Id).Do()
		checkErr("Trouble recieving message", err)
		bar.Add(1)
		// Filter out any message without headers
		if fullMsg.Payload.Headers == nil {
			continue
		} else {
			*msg = *fullMsg
		}
	}
}

// for i, header := range msg.Payload.Headers {
// 	if header.Name == "From" {
// 		// Filter out Hangouts messages
// 		if strings.Contains(msg.Payload.Headers[i].Value, "profiles.google.com") {
// 			break
// 		}

// 		// Add email sender to our tally
// 		_, exists := senderCount[msg.Payload.Headers[i].Value]
// 		if exists {
// 			senderCount[msg.Payload.Headers[i].Value]++
// 		} else {
// 			senderCount[msg.Payload.Headers[i].Value] = 1
// 		}
// 		// Stop iterating over headers
// 		break
// 	}
// }

// func sortSliceByIntVal(m map[string]int) []emailCount {
// 	var ss []emailCount
// 	for k, v := range m {
// 		ss = append(ss, emailCount{k, v})
// 	}

// 	sort.Slice(ss, func(i, j int) bool {
// 		return ss[i].Count > ss[j].Count
// 	})

// 	return ss
// }
