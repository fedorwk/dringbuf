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
	payload := flag.Int("payload", 0, "value size in bytes: 0 (int), 64, 1024, 4096 or 16384")
	flag.Parse()

	if *messages <= 0 || *capacity <= 0 || *reps <= 0 {
		fmt.Fprintln(os.Stderr, "messages, capacity and reps must all be positive")
		flag.Usage()
		os.Exit(2)
	}

	var handoff func(bench.Options, int) (time.Duration, int, bool)
	var payloadSize int
	switch *payload {
	case 0:
		handoff = bench.RunHandoff
	case bench.PayloadSize64:
		handoff = bench.RunPayloadHandoff64
		payloadSize = bench.PayloadSize64
	case bench.PayloadSize1K:
		handoff = bench.RunPayloadHandoff1K
		payloadSize = bench.PayloadSize1K
	case bench.PayloadSize4K:
		handoff = bench.RunPayloadHandoff4K
		payloadSize = bench.PayloadSize4K
	case bench.PayloadSize16K:
		handoff = bench.RunPayloadHandoff16K
		payloadSize = bench.PayloadSize16K
	default:
		fmt.Fprintln(os.Stderr, "payload must be 0, 64, 1024, 4096 or 16384")
		flag.Usage()
		os.Exit(2)
	}

	// For payloads, scale the message count so the total byte volume stays
	// comparable to the int case (8 bytes/value), keeping run times sane.
	effective := *messages
	if payloadSize > 0 {
		effective = max((*messages)*8/payloadSize, 10_000)
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
	label := "int (8 B)"
	if payloadSize > 0 {
		label = comma(int64(payloadSize)) + " B"
	}
	fmt.Fprintf(w, "goroutine handoff: %s values of %s, capacity=%d, %d runs (best)\n",
		comma(int64(effective)), label, *capacity, *reps)
	fmt.Fprintln(w, "variant\t ns/op\t msg/s\t delivered\t valid")
	fmt.Fprintln(w, "-----\t -----\t -----\t ---------\t -----")

	for _, v := range variants {
		opts := bench.Options{Kind: v.kind, Capacity: *capacity}

		var best time.Duration
		received := 0
		ok := true
		for range *reps {
			elapsed, n, runOK := handoff(opts, effective)
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
