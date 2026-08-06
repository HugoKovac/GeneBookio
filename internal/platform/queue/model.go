package queue

type ConfigQueue struct {
	USER     string `envconfig:"RABBITMQ_DEFAULT_USER"`
	PASSWORD string `envconfig:"RABBITMQ_DEFAULT_PASS"`
	HOST     string `envconfig:"RABBITMQ_HOST"`
}
