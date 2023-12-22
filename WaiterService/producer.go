package main

type Producer interface {
	send(message *Message, destination string) error
	connect() error
	close() error
}
