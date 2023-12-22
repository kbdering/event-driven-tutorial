package main

type Servicer interface {
	processMessage(message *Message) (*Message, error)
	setup() error
	tearDown() error
	run()
}

type Service struct {
	chRequests  chan *Message
	chResponses chan *Message
	threads     int
}
