KEYS_DIR?=keys/
KEY_NAME?=JWT_

ent:
	go generate ./pkg/ent

api:
	go run cmd/api/main.go

epub_split:
	go run cmd/epub_parser/main.go ${ARG}

script:
	go run cmd/script/main.go ${ARG}

docker:
	docker compose up -d

genNewKeys:
	mkdir -p keys
	openssl genpkey -algorithm RSA -out ${KEYS_DIR}${KEY_NAME}private.pem -pkeyopt rsa_keygen_bits:2048
	openssl rsa -pubout -in ${KEYS_DIR}${KEY_NAME}private.pem -out ${KEYS_DIR}${KEY_NAME}public.pem
