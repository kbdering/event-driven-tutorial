package main

type Consumer interface {
	poll(timeout int) error
	connect() error
	close() error
}

