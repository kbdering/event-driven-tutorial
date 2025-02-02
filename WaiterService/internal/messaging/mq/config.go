package mq

type MQConfig struct {
	Host          string
	Port          int
	Username      string
	Password      string
	RequestQueue  string
	ResponseQueue string
	MaxRetries    int
	RetryBackoff  int
}
