package main

import (
	"os"

	"ssch.cc/podtrawl/internal/app"
)

func main() {
	os.Exit(app.CLI(os.Args[1:]))
}
