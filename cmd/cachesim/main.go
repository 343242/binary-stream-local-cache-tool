package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"fastReadFile/internal/simtool"
)

func main() {
	root := flag.String("root", "", "empty or nonexistent workspace root")
	profileName := flag.String("profile", "", "simulator profile: medium or large")
	batchSize := flag.Int("batch-size", simtool.DefaultBatchSize, "records per WriteBatch call")
	flag.Parse()

	if *root == "" {
		exitErr(fmt.Errorf("--root is required"))
	}
	profile, err := simtool.ResolveProfile(*profileName)
	if err != nil {
		exitErr(err)
	}
	report, err := simtool.Run(context.Background(), simtool.RunConfig{
		RootDir:   *root,
		Profile:   profile,
		BatchSize: *batchSize,
	})
	if err != nil {
		exitErr(err)
	}
	fmt.Print(simtool.FormatReport(report))
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
