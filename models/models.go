package models

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
	URL string `json:"url,omitempty"`
}

type MessageRequestFacebook struct {
	Recipient struct {
		ID string `json:"id"`
	} `json:"recipient"`
	Message MessageRequestContent `json:"message"`
}

type MessageRequestContent struct {
	Text       string     `json:"text,omitempty"`
	Attachment Attachment `json:"attachment,omitempty"`
}

type Recipient struct {
	ID string `json:"id"`
}

type MessageContent struct {
	Text string `json:"text"`
}

type MessageRequestFacebookMedia struct {
	Recipient Recipient      `json:"recipient"`
	Message   MessageContent `json:"message"`
}
