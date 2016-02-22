package main

import (
	"flag"
	log "github.com/Sirupsen/logrus"
	"github.com/bamstats"
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

var (
	cpu        = flag.Int("cpu", 1, "number of cpus to be used")
	bam        = flag.String("bam", "", "file to read")
	annotation = flag.String("annotation", "", "bgzip compressed and indexed annotation file")
	loglevel   = flag.String("loglevel", "warn", "logging level")
)

func main() {
	flag.Parse()
	level, err := log.ParseLevel(*loglevel)
	check(err)
	log.SetLevel(level)
	if *bam == "" {
		log.Fatal("no file specified")
	}
	stats := bamstats.Coverage(*bam, *annotation, *cpu)
	bamstats.OutputJson(stats)
}
