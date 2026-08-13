package database

type ConfigDB struct {
	USER          string `envconfig:"MYSQL_USER"`
	PASSWORD      string `envconfig:"MYSQL_PASSWORD"`
	ROOT_PASSWORD string `envconfig:"MYSQL_ROOT_PASSWORD"`
	DATABASE      string `envconfig:"MYSQL_DATABASE"`
	HOST          string `envconfig:"MYSQL_HOST"`
}
