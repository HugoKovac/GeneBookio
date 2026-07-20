ent:
	go generate ./pkg/ent

api:
	go run cmd/api/main.go

docker:
	docker compose up -d