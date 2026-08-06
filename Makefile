KEYS_DIR?=keys/
KEY_NAME?=JWT_

ent:
	go generate ./pkg/ent

api:
	go run cmd/api/main.go

epub_split:
	go run cmd/epub_parser/main.go

prepare_chapters:
	go run cmd/prepare_chapters/main.go

upload_service:
	go run cmd/upload_book/main.go

generate_script:
	go run cmd/generate_script/main.go

docker:
	docker compose up -d

genNewKeys:
	mkdir -p keys
	openssl genpkey -algorithm RSA -out ${KEYS_DIR}${KEY_NAME}private.pem -pkeyopt rsa_keygen_bits:2048
	openssl rsa -pubout -in ${KEYS_DIR}${KEY_NAME}private.pem -out ${KEYS_DIR}${KEY_NAME}public.pem
