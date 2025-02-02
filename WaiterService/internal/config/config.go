package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Service ServiceConfig
	Kafka   KafkaConfig
	MQ      MQConfig `mapstructure:"mq"`
}

type ServiceConfig struct {
	NumThreads      int    `mapstructure:"num_threads"`
	WaitTime        int    `mapstructure:"wait_time"`
	RequestBuffer   int    `mapstructure:"request_buffer"`
	ResponseBuffer  int    `mapstructure:"response_buffer"`
	RateLimit       int    `mapstructure:"rate_limit"`
	NumProducers    int    `mapstructure:"num_producers"`
	NumConsumers    int    `mapstructure:"num_consumers"`
	PollDuration    int    `mapstructure:"poll_duration"`
	MessagingSystem string `mapstructure:"messaging_system"`
}

type KafkaConfig struct {
	BootstrapServers string `mapstructure:"bootstrap_servers"`
	ClientID         string `mapstructure:"client_id"`
	GroupID          string `mapstructure:"group_id"`
	RequestTopic     string `mapstructure:"request_topic"`
	ResponseTopic    string `mapstructure:"response_topic"`
	MaxRetries       int    `mapstructure:"max_retries"`
	RetryBackoff     int    `mapstructure:"retry_backoff"`
}

type MQConfig struct {
	Host          string `mapstructure:"host"`
	Port          int    `mapstructure:"port"`
	Username      string `mapstructure:"username"`
	Password      string `mapstructure:"password"`
	RequestQueue  string `mapstructure:"request_queue"`
	ResponseQueue string `mapstructure:"response_queue"`
	MaxRetries    int    `mapstructure:"max_retries"`
	RetryBackoff  int    `mapstructure:"retry_backoff"`
}

func (c *Config) Validate() error {
	if c.Service.MessagingSystem != "kafka" && c.Service.MessagingSystem != "mq" {
		return fmt.Errorf("unsupported messaging system: %s", c.Service.MessagingSystem)
	}

	if c.Service.NumProducers < 1 {
		return fmt.Errorf("num_producers must be at least 1")
	}

	if c.Service.NumConsumers < 1 {
		return fmt.Errorf("num_consumers must be at least 1")
	}

	return nil
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}
