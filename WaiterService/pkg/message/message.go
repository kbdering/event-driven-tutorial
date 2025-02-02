package message

import (
	"time"
	"golang.org/x/exp/maps"
)

type Message struct {
	Headers MessageHeaders
	Body    []byte
	Timestamp time.Time
}


type MessageHeaders struct {
	headers map[string][]byte
}



func (m MessageHeaders) Set(key string, value string) {
	m.headers[key] = []byte(value)
}

func (m MessageHeaders) Get(key string) string {
	return string(m.headers[key])
}

func (m MessageHeaders) Keys() []string {
	return maps.Keys(m.headers)
}

func NewMessageHeaders() * MessageHeaders {
	var h MessageHeaders
	h.headers = make(map[string][]byte)
	return &h
}

func NewMessage() * Message {
	var m Message
	m.Headers = *NewMessageHeaders()
	m.Body = make([]byte,0)
	return &m
}

