package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/joho/godotenv"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
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
	MID  string `json:"mid"`
	Text string `json:"text"`
}

type MessageRequestFacebook struct {
	Recipient struct {
		ID string `json:"id"`
	} `json:"recipient"`
	Message struct {
		Text string `json:"text"`
	} `json:"message"`
}

func sendMessageFacebook(senderID, messageText string) error {

	if len(messageText) == 0 {
		return errors.New("message can't be empty")
	}

	messagePayload := MessageRequestFacebook{
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
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("Failed to read body: %v", err)
		c.String(http.StatusInternalServerError, "Internal Server Error")
		return
	}

	log.Printf("Received body: %s", string(body))

	var message MessageFB
	if err := json.Unmarshal(body, &message); err != nil {
		log.Printf("Failed to unmarshal body: %v", err)
		c.String(http.StatusBadRequest, "Bad Request")
		return
	}

	if len(message.Entry) == 0 || len(message.Entry[0].Messaging) == 0 {
		log.Printf("No messaging content received")
		c.String(http.StatusBadRequest, "Bad Request")
		return
	}

	receivedMessage := message.Entry[0].Messaging[0].Message.Text
	log.Printf("Decoded message: %s", receivedMessage)

	err = sendMessageFacebook(message.Entry[0].Messaging[0].Sender.ID, "Automatically Reply 🙌🏻")
	if err != nil {
		log.Printf("Failed to send message: %v", err)
		c.String(http.StatusInternalServerError, "Internal Server Error")
		return
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
