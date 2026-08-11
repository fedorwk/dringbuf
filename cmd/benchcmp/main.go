package main

import (
	"dringbuf/internal/bench"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"
)

func main() {
	messages := flag.Int("messages", 1_000_000, "number of values transferred per handoff")
	capacity := flag.Int("capacity", 256, "channel/buffer capacity")
	reps := flag.Int("reps", 5, "runs per variant; best is reported")
	flag.Parse()

	if *messages <= 0 || *capacity <= 0 || *reps <= 0 {
		fmt.Fprintln(os.Stderr, "messages, capacity and reps must all be positive")
		flag.Usage()
		os.Exit(2)
	}

	type variant struct {
		name string
		kind bench.Kind
	}
	variants := []variant{
		{"chan unbuffered", bench.ChanUnbuffered},
		{"chan buffered", bench.ChanBuffered},
		{"stream block", bench.StreamBlock},
		{"stream drop-oldest", bench.StreamDropOldest},
		{"stream drop-newest", bench.StreamDropNewest},
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintf(w, "goroutine handoff: %s values, capacity=%d, %d runs (best)\n",
		comma(int64(*messages)), *capacity, *reps)
	fmt.Fprintln(w, "variant\t ns/op\t msg/s\t delivered\t valid")
	fmt.Fprintln(w, "-----\t -----\t -----\t ---------\t -----")

	for _, v := range variants {
		opts := bench.Options{Kind: v.kind, Capacity: *capacity}

		var best time.Duration
		received := 0
		ok := true
		for range *reps {
			elapsed, n, runOK := bench.RunHandoff(opts, *messages)
			ok = ok && runOK
			received = n
			if best == 0 || elapsed < best {
				best = elapsed
			}
		}

		status := "OK"
		if !ok {
			status = "FAIL"
		}
		// ns/op is per delivered value, so msg/s is the true throughput of
		// messages passed; drop strategies deliver fewer than requested.
		nsOp := float64(best.Nanoseconds()) / float64(received)

		fmt.Fprintf(w, "%s\t %.1f\t %s/s\t %s\t %s\n",
			v.name, nsOp, comma(int64(1e9/nsOp)), comma(int64(received)), status)
	}
	w.Flush()
}

func comma(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	out := make([]byte, 0, len(s)+len(s)/3)
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(c))
	}
	return string(out)
}
