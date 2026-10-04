// Command input-repeat 跑按键连发样例。
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"example.com/repeat"
)

// Run 执行一次命令行调用，返回退出码。
func Run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("input-repeat", flag.ContinueOnError)
	flags.SetOutput(stderr)
	sample := flags.String("sample", "delay", "delay / cap / release / outoforder / burst / idem / scan")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *sample == "delay" {
		r := repeat.New(400, 100, 3, 100)
		r.Submit(repeat.Event{ID: "e1", Key: "space", AtMs: 0, Down: true})
		fmt.Fprintf(stdout, "keys=%d\n", len(r.Tick(100)))
		return 0
	}
	if *sample == "cap" {
		r := repeat.New(100, 100, 2, 100)
		r.Submit(repeat.Event{ID: "e1", Key: "a", AtMs: 0, Down: true})
		emitted := 0
		for now := 100; now <= 400; now += 100 {
			emitted += len(r.Tick(now))
		}
		fmt.Fprintf(stdout, "repeats=%d\n", emitted)
		return 0
	}
	if *sample == "release" {
		r := repeat.New(400, 100, 2, 100)
		r.Submit(repeat.Event{ID: "e1", Key: "a", AtMs: 0, Down: true})
		r.Tick(400)
		r.Submit(repeat.Event{ID: "e2", Key: "a", AtMs: 500, Down: false})
		r.Submit(repeat.Event{ID: "e3", Key: "a", AtMs: 600, Down: true})
		fmt.Fprintf(stdout, "keys=%d\n", len(r.Tick(1200)))
		return 0
	}
	if *sample == "outoforder" {
		r := repeat.New(100, 100, 10, 100)
		r.Submit(repeat.Event{ID: "e1", Key: "a", AtMs: 0, Down: true})
		r.Submit(repeat.Event{ID: "e2", Key: "b", AtMs: 5000, Down: true})
		r.Submit(repeat.Event{ID: "e3", Key: "a", AtMs: 200, Down: false})
		fmt.Fprintf(stdout, "keys=%d\n", len(r.Tick(300)))
		return 0
	}
	if *sample == "burst" {
		r := repeat.New(100, 100, 10, 4)
		r.Submit(repeat.Event{ID: "e1", Key: "a", AtMs: 0, Down: true})
		keys := r.Tick(1000)
		fmt.Fprintf(stdout, "keys=%d dropped=%d\n", len(keys), r.Dropped())
		return 0
	}
	if *sample == "idem" {
		r := repeat.New(400, 100, 3, 100)
		r.Submit(repeat.Event{ID: "e1", Key: "a", AtMs: 0, Down: true})
		second := r.Submit(repeat.Event{ID: "e1", Key: "a", AtMs: 100, Down: false})
		fmt.Fprintf(stdout, "second=%v keys=%d\n", second, len(r.Tick(400)))
		return 0
	}
	if *sample == "scan" {
		r := repeat.New(400, 100, 1000000, 1000000)
		for index := 0; index < 3000; index++ {
			r.Submit(repeat.Event{
				ID:   fmt.Sprintf("e-%d", index),
				Key:  fmt.Sprintf("key-%d", index),
				AtMs: 0,
				Down: true,
			})
		}
		for index := 0; index < 3000; index++ {
			r.Tick(400 + index)
		}
		fmt.Fprintf(stdout, "scanned=%d\n", r.Scanned())
		return 0
	}
	fmt.Fprintln(stderr, "需要 --sample delay|cap|release|outoforder|burst|idem|scan")
	return 2
}

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}
