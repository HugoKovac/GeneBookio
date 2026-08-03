package queue

type ConfigQueue struct {
	USER     string `envconfig:"RABBITMQ_DEFAULT_USER"`
	PASSWORD string `envconfig:"RABBITMQ_DEFAULT_PASS"`
}
