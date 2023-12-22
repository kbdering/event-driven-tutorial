package main

import (
	"fmt"
	"time"
)

type WaiterService struct {
	Service
	waitTime rune
}

func (ws *WaiterService) processMessage(message *Message) (*Message, error) {
	_, span := getTracer().Start(getContextFromHeaders(message), "wait")
	defer span.End()
	time.Sleep(time.Duration(ws.waitTime) * time.Millisecond)

	resp := message
	resp.body = []byte(fmt.Sprintf("Hello %s, i've waited for you for %d milliseconds", string(message.body), ws.waitTime))
	fmt.Println(string(resp.body))
	return resp, nil

}

func (ws *WaiterService) run() {

	for i := 0; i < ws.threads; i++ {
		go func() {
			for {
				message := <-ws.chRequests
				response, err := ws.processMessage(message)

				if err == nil {
					ws.chResponses <- response
				}
				fmt.Println(ws.chResponses)
				fmt.Println(err)
			}
		}()
	}
}

func (ws *WaiterService) tearDown() error {
	return nil
}

func (ws *WaiterService) setup() error {
	return nil
}
