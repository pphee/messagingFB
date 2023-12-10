package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type MessageFB struct {
	Object string `json:"object"`
	Entry  []Entry
}

type Entry struct {
	ID        string      `json:"id"`
	Time      int64       `json:"time"`
	Messaging []Messaging `json:"messaging"`
}

type Messaging struct {
	Sender    User    `json:"sender"`
	Recipient User    `json:"recipient"`
	Timestamp int64   `json:"timestamp"`
	Message   Message `json:"message"`
}

type User struct {
	ID string `json:"id"`
}

type Message struct {
	MID         string       `json:"mid"`
	Text        string       `json:"text,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	IsEcho      bool         `json:"is_echo,omitempty"`
}

type Attachment struct {
	Type    string  `json:"type"`
	Payload Payload `json:"payload"`
}

type Payload struct {
	URL string `json:"url,omitempty"` // For media files like audio, image, file
}

type MessageRequestFacebook struct {
	Recipient struct {
		ID string `json:"id"`
	} `json:"recipient"`
	Message MessageRequestContent `json:"message"`
}

type MessageRequestContent struct {
	Text       string     `json:"text,omitempty"`
	Attachment Attachment `json:"attachment,omitempty"` // Change from Attachments to Attachment
}

type MessageRequestFacebookMedia struct {
	Recipient struct {
		ID string `json:"id"`
	} `json:"recipient"`
	Message struct {
		Text string `json:"text"`
	} `json:"message"`
}

// sendMessageToFacebook sends a message or attachment to a Facebook user.
func sendMessageToFacebook(senderID string, messageContent MessageRequestContent) error {
	messagePayload := MessageRequestFacebook{
		Recipient: struct {
			ID string `json:"id"`
		}{ID: senderID},
		Message: messageContent,
	}

	payloadBytes, err := json.Marshal(messagePayload)
	if err != nil {
		return fmt.Errorf("error marshalling request: %w", err)
	}

	log.Printf("Sending message to Facebook with payload: %s", string(payloadBytes))
	return sendRequestToFacebook(payloadBytes)
}

func sendImageToFacebook(senderID, imageURL string) error {
	attachment := Attachment{
		Type:    "image",
		Payload: Payload{URL: imageURL},
	}

	messageContent := MessageRequestContent{
		Attachment: attachment,
	}

	return sendMessageToFacebook(senderID, messageContent)
}

func sendAudioToFacebook(senderID, audioURL string) error {
	attachment := Attachment{
		Type:    "audio",
		Payload: Payload{URL: audioURL},
	}

	messageContent := MessageRequestContent{
		Attachment: attachment,
	}

	return sendMessageToFacebook(senderID, messageContent)
}

func sendVideoToFacebook(senderID, videoURL string) error {
	attachment := Attachment{
		Type:    "video",
		Payload: Payload{URL: videoURL},
	}

	messageContent := MessageRequestContent{
		Attachment: attachment,
	}

	return sendMessageToFacebook(senderID, messageContent)
}

func sendFileToFacebook(senderID, fileURL string) error {
	attachment := Attachment{
		Type:    "file",
		Payload: Payload{URL: fileURL},
	}

	messageContent := MessageRequestContent{
		Attachment: attachment,
	}

	return sendMessageToFacebook(senderID, messageContent)
}

// sendMessageFacebook sends a message to a Facebook user.
func sendRequestToFacebook(payloadBytes []byte) error {
	fmt.Println("payloadBytes", bytes.NewBuffer(payloadBytes))
	requestURL := fmt.Sprintf("%s/me/messages?access_token=%s", os.Getenv("GRAPHQL_URL"), os.Getenv("ACCESS_TOKEN"))
	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed: status %v, body: %s", resp.Status, string(body))
	}

	return nil
}

func sendMessageFacebook(senderID, messageText string) error {

	if len(messageText) == 0 {
		return errors.New("message can't be empty")
	}

	messagePayload := MessageRequestFacebookMedia{
		Recipient: struct {
			ID string `json:"id"`
		}{
			ID: senderID,
		},
		Message: struct {
			Text string `json:"text"`
		}{
			Text: messageText,
		},
	}

	fmt.Println("messagePayload", messagePayload)

	payloadBytes, err := json.Marshal(messagePayload)
	if err != nil {
		return fmt.Errorf("error marshalling request: %w", err)
	}

	requestURL := fmt.Sprintf("%s/me/messages?access_token=%s", os.Getenv("GRAPHQL_URL"), os.Getenv("ACCESS_TOKEN"))
	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	log.Printf("Message sent; response status: %v", resp.Status)

	return nil
}

// verifyWebhook verifies the webhook with Facebook.
func verifyWebhook(c *gin.Context) {
	verifyToken := c.Query("hub.verify_token")
	if verifyToken != os.Getenv("VERIFY_TOKEN") {
		log.Printf("Invalid verification token: %s", verifyToken)
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	challenge := c.Query("hub.challenge")
	c.String(http.StatusOK, challenge)
}

// handleWebhookEvent processes incoming webhook events from Facebook.
func handleWebhookEvent(c *gin.Context) {
	var message MessageFB
	if err := json.NewDecoder(c.Request.Body).Decode(&message); err != nil {
		log.Printf("Error decoding message: %v", err)
		c.String(http.StatusBadRequest, "Bad Request")
		return
	}

	for _, entry := range message.Entry {
		for _, messaging := range entry.Messaging {
			if messaging.Message.IsEcho {
				log.Printf("Ignoring echo message")
				continue
			}

			senderID := messaging.Sender.ID

			// Check if the message is a text message
			if messaging.Message.Text != "" {
				textMessage := "Received your message: " + messaging.Message.Text
				if err := sendMessageFacebook(senderID, textMessage); err != nil {
					log.Printf("Failed to send message: %v", err)
					continue
				}

			}
			// Handle attachments
			for _, attachment := range messaging.Message.Attachments {
				switch attachment.Type {
				case "image":
					imageURL := "https://i.gifer.com/Ifph.gif"
					if err := sendImageToFacebook(senderID, imageURL); err != nil {
						log.Printf("Failed to send image: %v", err)
						continue
					}
				case "audio":
					audioURL := attachment.Payload.URL
					if err := sendAudioToFacebook(senderID, audioURL); err != nil {
						log.Printf("Failed to send audio: %v", err)
						continue
					}
				case "video":
					videoURL := attachment.Payload.URL
					if err := sendVideoToFacebook(senderID, videoURL); err != nil {
						log.Printf("Failed to send video: %v", err)
						continue
					}
				case "file":
					file := attachment.Payload.URL
					if err := sendFileToFacebook(senderID, file); err != nil {
						log.Printf("Failed to send file: %v", err)
						continue
					}
				default:
					log.Printf("Received an unsupported attachment: %s", attachment.Type)
				}
			}
		}
	}

	c.Status(http.StatusOK)
}

func webhookHandler(c *gin.Context) {
	if c.Request.Method == http.MethodGet {
		verifyWebhook(c)
	} else if c.Request.Method == http.MethodPost {
		handleWebhookEvent(c)
	} else {
		log.Printf("Invalid method: not GET or POST")
		c.AbortWithStatus(http.StatusMethodNotAllowed)
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	router := gin.Default()

	router.Any("/", webhookHandler)

	log.Fatal(router.Run(":8080"))
}
