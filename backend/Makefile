KEYS_DIR?=keys/
KEY_NAME?=JWT_
TEST_MODE?=true

ent:
	go generate ./pkg/ent/generate.go

api:
	go run cmd/api/main.go

epub_split:
	go run cmd/epub_parser/main.go

# TEST_MODE=true swaps the real OpenAI client for a substitution client that
# echoes the request back wrapped in ==START/END OF REQUEST== markers, so the
# pipeline can be exercised without spending real AI tokens.
prepare_chapters:
	AI_TEST_MODE=$(TEST_MODE) go run cmd/prepare_chapters/main.go

admin:
	go run cmd/admin/main.go

generate_script:
	AI_TEST_MODE=$(TEST_MODE) go run cmd/generate_script/main.go

generate_tts:
	AI_TEST_MODE=$(TEST_MODE) go run cmd/generate_tts/main.go

swag:
	swag init -g cmd/api/main.go -o docs/

docker-hybrid:
	docker compose up -d db rabbitmq seaweed

docker:
	docker compose up -d

genNewKeys:
	mkdir -p keys
	openssl genpkey -algorithm RSA -out ${KEYS_DIR}${KEY_NAME}private.pem -pkeyopt rsa_keygen_bits:2048
	openssl rsa -pubout -in ${KEYS_DIR}${KEY_NAME}private.pem -out ${KEYS_DIR}${KEY_NAME}public.pem

.PHONY: api
