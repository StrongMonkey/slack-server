package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

// SlackCommandPayload represents the incoming Slack slash command payload
type SlackCommandPayload struct {
	Text        string `json:"text"`
	ChannelID   string `json:"channel_id"`
	ThreadTS    string `json:"thread_ts"` // Will be empty if not in a thread
	ResponseURL string `json:"response_url"`
	UserID      string `json:"user_id"` // Add this field
}

// APIRequestBody represents the body we'll send to the Acorn API
type APIRequestBody struct {
	THREAD_ID  string `json:"THREAD_ID"`
	CHANNEL_ID string `json:"CHANNEL_ID"`
	USER_ID    string `json:"USER_ID"`
	QUERY      string `json:"QUERY"`
}

// Add these new types for Slack Events
type SlackEventPayload struct {
	Type      string            `json:"type"`
	Challenge string            `json:"challenge"`
	Event     SlackMessageEvent `json:"event"`
}

type SlackMessageEvent struct {
	Type      string `json:"type"`
	Text      string `json:"text"`
	ChannelID string `json:"channel"`
	ThreadTS  string `json:"thread_ts,omitempty"`
	TS        string `json:"ts"`
	User      string `json:"user"`
}

func main() {
	// Check for required environment variable
	accessToken := os.Getenv("OBOT_ACCESS_TOKEN")
	if accessToken == "" {
		log.Fatal("OBOT_ACCESS_TOKEN environment variable must be set")
	}

	http.HandleFunc("/slack/command", handleSlackCommand(accessToken))
	http.HandleFunc("/slack/events", handleSlackEvents(accessToken))

	port := "8088"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	fmt.Printf("Server starting on port %s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func handleSlackCommand(accessToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse the form data from Slack
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form data", http.StatusBadRequest)
			return
		}

		// Extract values from the Slack command
		slackPayload := SlackCommandPayload{
			Text:      r.FormValue("text"),
			ChannelID: r.FormValue("channel_id"),
			ThreadTS:  r.FormValue("thread_ts"), // Will be empty if not in a thread
			UserID:    r.FormValue("user_id"),
		}

		// Prepare the request body for the Acorn API
		apiBody := APIRequestBody{
			THREAD_ID:  slackPayload.ThreadTS,
			CHANNEL_ID: slackPayload.ChannelID,
			USER_ID:    slackPayload.UserID,
			QUERY:      slackPayload.Text,
		}

		// Convert the body to JSON
		jsonBody, err := json.Marshal(apiBody)
		if err != nil {
			http.Error(w, "Failed to create request body", http.StatusInternalServerError)
			return
		}

		// Create the request to the Acorn API
		apiURL := "https://main.acornlabs.com/api/assistants/a1gnhpr/projects/p144lqv/tasks/w126clz/run?step=*"
		req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(jsonBody))
		if err != nil {
			http.Error(w, "Failed to create API request", http.StatusInternalServerError)
			return
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Cookie", "obot_access_token="+accessToken)

		// Make the request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, "Failed to send request to API", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// Respond to Slack immediately to avoid timeout
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"response_type": "in_channel"}`))
	}
}

func handleSlackEvents(accessToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Read the request body
		var payload SlackEventPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Failed to parse request body", http.StatusBadRequest)
			return
		}

		// Handle URL verification challenge
		if payload.Type == "url_verification" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"challenge": payload.Challenge,
			})
			return
		}

		// Handle message events
		if payload.Type == "event_callback" && payload.Event.Type == "app_mention" {
			// Prepare the request body for the Acorn API
			apiBody := APIRequestBody{
				THREAD_ID:  payload.Event.ThreadTS,
				CHANNEL_ID: payload.Event.ChannelID,
				USER_ID:    payload.Event.User,
				QUERY:      payload.Event.Text,
			}

			// If message is not in a thread, use the message TS as thread ID
			if apiBody.THREAD_ID == "" {
				apiBody.THREAD_ID = payload.Event.TS
			}

			// Convert the body to JSON
			jsonBody, err := json.Marshal(apiBody)
			if err != nil {
				http.Error(w, "Failed to create request body", http.StatusInternalServerError)
				return
			}

			// Create the request to the Acorn API
			apiURL := "https://main.acornlabs.com/api/assistants/a1gnhpr/projects/p144lqv/tasks/w1sbtgq/run?step=*"
			req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(jsonBody))
			if err != nil {
				http.Error(w, "Failed to create API request", http.StatusInternalServerError)
				return
			}

			// Set headers
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Cookie", "obot_access_token="+accessToken)

			// Make the request
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				http.Error(w, "Failed to send request to API", http.StatusInternalServerError)
				return
			}
			// Read and print the response body
			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				http.Error(w, "Failed to read response body", http.StatusInternalServerError)
				return
			}
			log.Printf("Response from API: %s", string(respBody))
			defer resp.Body.Close()
		}

		// Respond with 200 OK for all event callbacks
		w.WriteHeader(http.StatusOK)
	}
}
