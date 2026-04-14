package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"fastReadFile/internal/ops"
)

func main() {
	root := "."
	args := os.Args[1:]
	for len(args) >= 2 && args[0] == "--root" {
		root = args[1]
		args = args[2:]
	}
	if len(args) == 0 {
		exitErr(fmt.Errorf("missing subcommand"))
	}

	var (
		output string
		err    error
	)

	switch args[0] {
	case "stats":
		fs := flag.NewFlagSet("stats", flag.ExitOnError)
		format := fs.String("format", "text", "output format")
		_ = fs.Parse(args[1:])
		output, err = ops.Stats(root, *format)
	case "inspect-wal":
		output, err = ops.InspectWAL(root)
	case "inspect-cursor":
		fs := flag.NewFlagSet("inspect-cursor", flag.ExitOnError)
		destination := fs.String("destination", "", "cursor destination")
		_ = fs.Parse(args[1:])
		output, err = ops.InspectCursor(root, *destination)
	case "close-check":
		output, err = ops.CloseCheck(root)
	case "verify":
		output, err = ops.Verify(root)
	case "repair-tail":
		fs := flag.NewFlagSet("repair-tail", flag.ExitOnError)
		segmentID := fs.String("segment", "", "segment id")
		_ = fs.Parse(args[1:])
		id, parseErr := strconv.ParseUint(*segmentID, 10, 64)
		if parseErr != nil {
			exitErr(parseErr)
		}
		output, err = ops.RepairTail(root, id)
	case "inspect-segment":
		fs := flag.NewFlagSet("inspect-segment", flag.ExitOnError)
		segmentID := fs.String("segment", "", "segment id")
		_ = fs.Parse(args[1:])
		id, parseErr := strconv.ParseUint(*segmentID, 10, 64)
		if parseErr != nil {
			exitErr(parseErr)
		}
		output, err = ops.InspectSegment(root, id)
	default:
		exitErr(fmt.Errorf("unknown subcommand %q", args[0]))
	}

	if err != nil {
		exitErr(err)
	}
	fmt.Println(output)
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
