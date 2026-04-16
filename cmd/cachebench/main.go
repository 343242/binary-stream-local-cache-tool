package main

import (
	"flag"
	"fmt"
	"os"

	"fastReadFile/internal/benchtool"
)

func main() {
	records := flag.Int("records", 10000, "record count")
	payloadBytes := flag.Int("payload-bytes", 32, "payload size in bytes")
	flag.Parse()

	report, err := benchtool.Run(*records, *payloadBytes)
	if err != nil {
		exitErr(err)
	}
	fmt.Print(benchtool.FormatReport(report))
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
