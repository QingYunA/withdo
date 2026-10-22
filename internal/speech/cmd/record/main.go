// Command record tries internal/speech on the command line: it records the microphone for a few
// seconds, prints the loudest level it saw and writes the WAV.
//
//	internal/speech/cmd/record/run.sh [-for 5s] [-o /tmp/record.wav]
//
// A bare binary is ended by macOS when it asks for the microphone, so run.sh wraps it in an app
// bundle with the usage description.
package main

import (
	"flag"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/QingYunA/adhd-todo/internal/speech"
)

func main() {
	dur := flag.Duration("for", 5*time.Second, "how long to record")
	out := flag.String("o", "", "write the WAV here")
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

	var peak atomic.Uint32
	err := r.Start(func(l float32) {
		for {
			old := peak.Load()
			if uint32(l*1000) <= old || peak.CompareAndSwap(old, uint32(l*1000)) {
				return
			}
		}
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "start:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "recording for %s; speak now\n", *dur)
	time.Sleep(*dur)
	wav, err := r.Stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "stop:", err)
		os.Exit(1)
	}
	fmt.Printf("recorded %d bytes (%.1f s), peak level %.2f\n", len(wav), float64(len(wav)-44)/32000, float64(peak.Load())/1000)
	if *out != "" {
		if err := os.WriteFile(*out, wav, 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("wrote", *out)
	}
}
