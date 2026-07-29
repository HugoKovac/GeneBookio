package config

import (
	"crypto/rsa"
	"fmt"
	"hkorpo/book/pkg/errorwrapper"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

func readKeys(privatePath, publicPath string) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateBytes, err := os.ReadFile(privatePath)
	if err != nil {
		return nil, nil, err
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateBytes)
	if err != nil {
		return nil, nil, err
	}

	publicBytes, err := os.ReadFile(publicPath)
	if err != nil {
		return nil, nil, err
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicBytes)
	if err != nil {
		return nil, nil, err
	}

	return privateKey, publicKey, nil
}

func Init() (*Config, error) {
	var config Config
	err := godotenv.Load("cmd/api/.env")
	if err != nil {
		return nil, errorwrapper.Wrap(fmt.Errorf("Error loading .env file"))
	}

	if err := envconfig.Process("", &config); err != nil {
		return nil, errorwrapper.Wrap(err)
	}

	config.ConfigJWT.PrivateKey, config.ConfigJWT.PublicKey, err = readKeys(config.ConfigJWT.PRIVATE_KEY_PATH, config.ConfigJWT.PUBLIC_KEY_PATH)
	config.ConfigJWT.RefreshPrivateKey, config.ConfigJWT.RefreshPublicKey, err = readKeys(config.ConfigJWT.PRIVATE_REFRESH_KEY_PATH, config.ConfigJWT.PUBLIC_REFRESH_KEY_PATH)

	return &config, nil
}
