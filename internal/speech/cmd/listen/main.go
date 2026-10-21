// Command listen tries internal/speech on the command line: it listens to the microphone and prints
// the partial and final text.
//
//	go run ./internal/speech/cmd/listen [-for 8s]
//
// On macOS the first run asks for the microphone and speech recognition; the prompts name the
// terminal app, which must be allowed in System Settings > Privacy & Security.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/QingYunA/adhd-todo/internal/speech"
)

func main() {
	dur := flag.Duration("for", 8*time.Second, "how long to listen")
	flag.Parse()

	r := speech.New()
	if ok, why := r.Available(); !ok {
		fmt.Fprintln(os.Stderr, "unavailable:", why)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "authorization:", r.Authorization())
	if r.Authorization() != speech.AuthAuthorized {
		done := make(chan speech.Authorization, 1)
		r.RequestAuthorization(func(a speech.Authorization) { done <- a })
		select {
		case a := <-done:
			fmt.Fprintln(os.Stderr, "authorization now:", a)
		case <-time.After(60 * time.Second):
			fmt.Fprintln(os.Stderr, "no answer to the permission prompt in 60 s")
			os.Exit(1)
		}
	}

	final := make(chan string, 1)
	err := r.Start(
		func(t string) { fmt.Printf("partial: %s\n", t) },
		func(t string) { final <- t },
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "listening for %s; speak now\n", *dur)
	time.Sleep(*dur)
	r.Stop()
	select {
	case t := <-final:
		fmt.Printf("final: %s\n", t)
		if e := r.LastError(); e != nil {
			fmt.Fprintln(os.Stderr, "error:", e)
			os.Exit(1)
		}
	case <-time.After(5 * time.Second):
		fmt.Fprintln(os.Stderr, "no final text in 5 s")
		os.Exit(1)
	}
}
