//go:build ignore
// +build ignore

package main

//go:generate go run -mod=mod generate.go

import (
	"errors"
	"fmt"
	"log"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
)

func main() {
	fmt.Println("Generating entc...")
	generateEntc()
	fmt.Println("Successfully generated entc!")
}

func generateEntc() {
	if err := entc.Generate("./schema", &gen.Config{
		Features: []gen.Feature{
			gen.FeatureUpsert,
			gen.FeatureSnapshot,
		},
	}); !errors.Is(err, nil) {
		log.Fatalf("Error: failed running ent codegen: %v", err)
	}
}
