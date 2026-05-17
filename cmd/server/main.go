package main

import (
	"log"

	"sakurairo-go/internal/app"
)

func main() {
	application, err := app.New()
	if err != nil {
		log.Fatal(err)
	}
	application.Run()
}
