package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/nlopes/slack"
)

func main() {
	token := os.Getenv("SLACK_TOKEN")
	fmt.Println("token : " + token)

	api := slack.New(token) 
	api.SetDebug(true)

	rtm := api.NewRTM()
	go rtm.ManageConnection()

	u := os.Getenv("API_USER")
	p := os.Getenv("API_PASSWD")

	apiurl := os.Getenv("API_URL")

	c := NewBasicAuthClient(u, p, apiurl)

	// s := NewBasicAuthClient(u, p)

Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			fmt.Print("Event Received: ")
			switch ev := msg.Data.(type) {
			case *slack.ConnectedEvent:
				fmt.Println("Connection counter:", ev.ConnectionCount)

			case *slack.MessageEvent:
				fmt.Printf("Message: %v\n", ev)
				info := rtm.GetInfo()
				prefix := fmt.Sprintf("<@%s> ", info.User.ID)

				if ev.User != info.User.ID && strings.HasPrefix(ev.Text, prefix) {
					respond(rtm, ev, prefix, &c)
				}

			case *slack.RTMError:
				fmt.Printf("Error: %s\n", ev.Error())

			case *slack.InvalidAuthEvent:
				fmt.Printf("Invalid credentials")
				break Loop

			default:
				//Take no action
			}
		}
	}
}

// Bot Logic.
func respond(rtm *slack.RTM, msg *slack.MessageEvent, prefix string, c *Client) {
	var response string
	text := msg.Text
	//  fmt.Println("Text: '" + text + "'")

	channel := msg.Channel
	//  fmt.Println("Channel '" + channnel + "'")

	text = strings.TrimPrefix(text, prefix)
	text = strings.TrimSpace(text)
	text = strings.ToLower(text)

	fmt.Println("-------------->channel: " + channel)

	// Channel IDs for the allowed access
	acceptedChannels := map[string]bool{
		"DC3DY481L": true,
		"GC1HNEDPT": true,
	}

	acceptedGreetings := map[string]bool{
		"what's up?": true,
		"hey!":       true,
		"yo":         true,
	}
	acceptedHowAreYou := map[string]bool{
		"how's it going?": true,
		"how are ya?":     true,
		"feeling okay?":   true,
	}

	acceptedDispachting := map[string]bool{
		"Let's dispatch!": true,
		"dispatch":        true,
		"d":               true,
	}

	// Prefix you can start a Incident request with
	acceptTicketRequest := map[string]bool{
		"ticket": true,
		"t":      true,
	}

	parts := strings.Split(text, " ")

	var i_nr string
	i_nr = "0"

	var err bool
	err = false
	if len(parts) == 2 {

		if (acceptTicketRequest[parts[0]] || acceptedDispachting[parts[0]]) && len(parts[1]) == 8 && strings.HasPrefix(parts[1], "2") {
			i_nr = parts[1]
			text = parts[0]
		} else {
			text = "how's it going?"
			i_nr = "0"
			err = true
		}
	} else {
		//text = "hey!"
		i_nr = "0"
		err = true
	}

	if !acceptedChannels[channel] {
		response = "Hi, I can help you with SAP CRM tickets, but I am not allowed to talk to you, sorry! Please contact @Buschky for more information."
		rtm.SendMessage(rtm.NewOutgoingMessage(response, msg.Channel))
		return
	}

	if acceptedGreetings[text] {
		response = "What's up buddy!?!?!"
		rtm.SendMessage(rtm.NewOutgoingMessage(response, msg.Channel))
	} else if acceptedHowAreYou[text] {

		response = "I don't like your input. Maybe you like to try it again? Use help for more information."
		rtm.SendMessage(rtm.NewOutgoingMessage(response, msg.Channel))
	} else if acceptedDispachting[text] {
		//response = "{	\"label\": \"Post this message on\",\"name\": \"channel_notify\",\"type\": \"select\",\"data_source\": \"conversations\"  }"
		//rtm.PostMessage("DC3DY481L", response)
		p := slack.PostMessageParameters{}
		a := []slack.AttachmentAction{slack.AttachmentAction{Name: "Press", Text: "Users", Type: "select", DataSource: "users"}}
		attachment := slack.Attachment{
			//Pretext: "Dispatching Mode",
			Text:    "to User",
			Actions: a, //[a]st
			// Uncomment the following part to send a field too
			/*
				Fields: []slack.AttachmentField{
					slack.AttachmentField{
						Title: "a",
						Value: "no",
					},
				},
			*/
		}

		p.Username = msg.Username
		p.User = msg.User
		//p.IconURL = msg.Icons.IconURL
		p.Attachments = []slack.Attachment{attachment}

		rtm.PostMessage(msg.Channel, "Dispatch Ticket "+i_nr, p)

	} else if acceptTicketRequest[text] && !err {
		//var i Incident
		fmt.Println("IncidentID :" + i_nr)

		var url string

		incident := Incident{}
		url = GetURLtoAPI(i_nr, "", c.APIurl)
		c.GetIncidentData(url, &incident)

		status := Status{}
		url = GetURLtoAPI(i_nr, "/Status", c.APIurl)
		c.GetIncidentData(url, &status)

		prio := Priority{}
		url = GetURLtoAPI(i_nr, "/Priority", c.APIurl)
		c.GetIncidentData(url, &prio)

		//fmt.println("AssignmentGroup " + i.AssignmentGroup)
		fmt.Println("Titel " + incident.D.Title)

		//fmt.Println("objid " + .i..ObjectID)
		//i, _ := client.GetIncident(1)
		response = "Info for Ticket " + incident.D.ObjectID
		response = response + "\n-> Title " + incident.D.Title
		response = response + "\n-> Status " + status.D.StatusDesc
		response = response + "\n-> Prio " + prio.D.PriorityDesc

		rtm.SendMessage(rtm.NewOutgoingMessage(response, msg.Channel))
	} else {
		response = "------------------------ "
		rtm.SendMessage(rtm.NewOutgoingMessage(response, msg.Channel))
	}

}

// Set the Client Structure to handle authentication
func NewBasicAuthClient(username, password string, url string) Client {
	return Client{
		Username: username,
		Password: password,
		APIurl:   url,
	}
}

// Structure of Client for HTTPS Authorisation
type Client struct {
	Username string
	Password string
	APIurl   string
}

// Autogenerated Structs from JSON File
// https://mholt.github.io/json-to-go/
type Status struct {
	D struct {
		Metadata struct {
			ID   string `json:"id"`
			URI  string `json:"uri"`
			Type string `json:"type"`
		} `json:"__metadata"`
		ProcessType string `json:"ProcessType"`
		StatusDesc  string `json:"StatusDesc"`
		StatusID    string `json:"StatusId"`
	} `json:"d"`
}

// Autogenerated Structs from JSON File
// https://mholt.github.io/json-to-go/
type Priority struct {
	D struct {
		Metadata struct {
			ID   string `json:"id"`
			URI  string `json:"uri"`
			Type string `json:"type"`
		} `json:"__metadata"`
		PriorityDesc string `json:"PriorityDesc"`
		PriorityID   string `json:"PriorityId"`
		IncidentSet  struct {
			Deferred struct {
				URI string `json:"uri"`
			} `json:"__deferred"`
		} `json:"IncidentSet"`
	} `json:"d"`
}

// Autogenerated Structs from JSON File
// https://mholt.github.io/json-to-go/
type Incident struct {
	D struct {
		Metadata struct {
			ID   string `json:"id"`
			URI  string `json:"uri"`
			Type string `json:"type"`
		} `json:"__metadata"`
		ObjectID          string `json:"ObjectId"`
		ProcessTypeID     string `json:"ProcessTypeId"`
		StatusID          string `json:"StatusId"`
		PriorityID        string `json:"PriorityId"`
		Title             string `json:"Title"`
		ResponsibleID     string `json:"ResponsibleId"`
		AssignmentGroupID string `json:"AssignmentGroupId"`
		AssignmentGroup   struct {
			Deferred struct {
				URI string `json:"uri"`
			} `json:"__deferred"`
		} `json:"AssignmentGroup"`
		ProcessType struct {
			Deferred struct {
				URI string `json:"uri"`
			} `json:"__deferred"`
		} `json:"ProcessType"`
		Status struct {
			Deferred struct {
				URI string `json:"uri"`
			} `json:"__deferred"`
		} `json:"Status"`
		Priority struct {
			Deferred struct {
				URI string `json:"uri"`
			} `json:"__deferred"`
		} `json:"Priority"`
		Responsible struct {
			Deferred struct {
				URI string `json:"uri"`
			} `json:"__deferred"`
		} `json:"Responsible"`
	} `json:"d"`
}

func GetURLtoAPI(id string, reqtype string, apiurl string) (url string) {
	url = fmt.Sprintf(apiurl + "/Incidents('" + id + "')" + reqtype + "?$format=json")
	fmt.Println("URL : " + url)
	return url
}
func (s *Client) GetIncidentData(url string, outData interface{}) error {

	//	url := getURLtoAPI(id, "/Status", s.APIurl)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	bytes, err := s.DoRequest(req)
	if err != nil {
		return err
	}

	err = json.Unmarshal(bytes, outData)
	if err != nil {
		return err
	}
	return nil
}

func (s *Client) DoRequest(req *http.Request) ([]byte, error) {
	req.SetBasicAuth(s.Username, s.Password)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if 200 != resp.StatusCode {
		return nil, fmt.Errorf("%s", body)
	}
	return body, nil
}
