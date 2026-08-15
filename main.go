package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"ssch.cc/podtrawl/config"
)

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	run(ctx)
}

func run(ctx context.Context) {
	conf, err := config.Get(nil)
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	} else {
		fmt.Println(conf)
	}
	fmt.Println("Waiting...")
	<-ctx.Done()
	fmt.Println("Got signal:", ctx.Err())
}
