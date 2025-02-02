package kafka

type KafkaConfig struct {
	BootstrapServers string
	ClientID         string
	GroupID          string
	RequestTopic     string
	ResponseTopic    string
	MaxRetries       int
	RetryBackoff     int
}
