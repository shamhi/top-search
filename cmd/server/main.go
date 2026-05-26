package main

import (
	"context"

	"github.com/shamhi/top-search/internal/app"
)

func main() {
	ctx := context.Background()

	a, err := app.New(ctx)
	if err != nil {
		panic("init failed: " + err.Error())
	}

	a.Run()
}
