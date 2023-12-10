package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"messaging/models"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func sendMediaToFacebook(senderID, mediaURL, mediaType string) error {
	payload := models.MessageRequestFacebook{
		Recipient: struct {
			ID string `json:"id"`
		}{ID: senderID},
		Message: models.MessageRequestContent{
			Attachment: models.Attachment{
				Type: mediaType,
				Payload: models.Payload{
					URL: mediaURL,
				},
			},
		},
	}

	return sendToFacebook(payload)
}

func sendToFacebook(payload interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshalling request: %w", err)
	}

	log.Printf("Sending payload to Facebook: %s", string(payloadBytes))

	return sendRequestToFacebook(payloadBytes)
}

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

func sendMessage(senderID, messageText string) error {
	if messageText == "" {
		return errors.New("message can't be empty")
	}

	payload := models.MessageRequestFacebookMedia{
		Recipient: models.Recipient{
			ID: senderID,
		},
		Message: models.MessageContent{
			Text: messageText,
		},
	}

	return sendToFacebook(payload)
}

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

func handleWebhookEvent(c *gin.Context) {
	var message models.MessageFB
	if err := json.NewDecoder(c.Request.Body).Decode(&message); err != nil {
		log.Printf("Error decoding message: %v", err)
		c.String(http.StatusBadRequest, "Bad Request")
		return
	}

	fmt.Println("-----------------------message-----------------------", message)

	for _, entry := range message.Entry {
		for _, messaging := range entry.Messaging {
			if messaging.Message.IsEcho {
				log.Printf("Ignoring echo message")
				continue
			}

			senderID := messaging.Sender.ID

			if messaging.Message.Text != "" {
				textMessage := "🖕🏻🖕🏼🖕🏽🖕🏾🖕🏿:  " + messaging.Message.Text
				if err := sendMessage(senderID, textMessage); err != nil {
					log.Printf("Failed to send message: %v", err)
					continue
				}

			}

			for _, attachment := range messaging.Message.Attachments {
				switch attachment.Type {
				case "image":
					imageURL := "https://i.gifer.com/Ifph.gif"
					if err := sendMediaToFacebook(senderID, imageURL, attachment.Type); err != nil {
						log.Printf("Failed to send image: %v", err)
						continue
					}
				case "audio":
					audioURL := attachment.Payload.URL
					if err := sendMediaToFacebook(senderID, audioURL, attachment.Type); err != nil {
						log.Printf("Failed to send audio: %v", err)
						continue
					}
				case "video":
					videoURL := attachment.Payload.URL
					if err := sendMediaToFacebook(senderID, videoURL, attachment.Type); err != nil {
						log.Printf("Failed to send video: %v", err)
						continue
					}
				case "file":
					file := attachment.Payload.URL
					if err := sendMediaToFacebook(senderID, file, attachment.Type); err != nil {
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
