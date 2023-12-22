package main

import "fmt"

func main() {

	initOpenTelemetry()

	defer closeOpenTelemetry()
	numProducers := 10
	numConsumers := 10
	numThreads := 50

	requestTopic := "Waiter-Requests"
	responseTopic := "Waiter-Responses"
	requestsChannel := make(chan *Message,10)
	responsesChannel := make(chan *Message, 10)

	pConfig := map[string]string{
		"bootstrap.servers": "localhost:29092",
		"client.id":         "golang",
	}

	cConfig := map[string]string{
		"bootstrap.servers": "localhost:29092",
		"client.id":         "golang",
		"group.id":          "waiter",
		"fetch.max.bytes":   "1",
		"topic":             requestTopic,
	}

	for i := 0; i < numProducers; i++ {

		go func() {
			producer := newKafkaProducer(pConfig, responsesChannel)
			producer.connect()
			defer producer.close()
			for {
				m := <-producer.chResponses
				producer.send(m, responseTopic)
				fmt.Println("sending response!")
			}
		}()

	}

	service := &WaiterService{
		Service:  Service{threads: numThreads, chRequests: requestsChannel, chResponses: responsesChannel},
		waitTime: 500,
	}
	service.setup()
	service.run()

	for i := 0; i < numConsumers; i++ {

		consumer, err := newKafkaConsumer(cConfig, requestsChannel)
		if err != nil {
			panic(err)
		}
		consumer.connect()
		go func() {
			run := true
			for run {
				err = consumer.poll(200)
			}
		}()
	}

	chWait := make(chan int)
	<-chWait

}
