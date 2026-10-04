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
	sample := flags.String("sample", "delay", "delay / cap / release / burst / scan")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *sample == "delay" {
		r := repeat.New(400, 100, 3)
		r.Press("space", 0)
		fmt.Fprintf(stdout, "keys=%d\n", len(r.Tick(100)))
		return 0
	}
	if *sample == "cap" {
		r := repeat.New(100, 100, 2)
		r.Press("a", 0)
		emitted := 0
		for now := 100; now <= 400; now += 100 {
			emitted += len(r.Tick(now))
		}
		fmt.Fprintf(stdout, "repeats=%d\n", emitted)
		return 0
	}
	if *sample == "release" {
		r := repeat.New(400, 100, 10)
		r.Press("a", 0)
		r.Tick(400)
		r.Release("a", 500)
		r.Press("a", 600)
		fmt.Fprintf(stdout, "keys=%d\n", len(r.Tick(700)))
		return 0
	}
	if *sample == "burst" {
		r := repeat.New(100, 100, 10)
		r.Press("a", 0)
		fmt.Fprintf(stdout, "keys=%d\n", len(r.Tick(1000)))
		return 0
	}
	if *sample == "scan" {
		r := repeat.New(400, 100, 1000000)
		for index := 0; index < 3000; index++ {
			r.Press(fmt.Sprintf("key-%d", index), 0)
		}
		for index := 0; index < 3000; index++ {
			r.Tick(400 + index)
		}
		fmt.Fprintf(stdout, "scanned=%d\n", r.Scanned())
		return 0
	}
	fmt.Fprintln(stderr, "需要 --sample delay|cap|release|burst|scan")
	return 2
}

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}
