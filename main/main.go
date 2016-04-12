package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/bamstats"
	"github.com/codegangsta/cli"
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

var (
	bam, annotation, loglevel string
	cpu, maxBuf               int
)

func run(c *cli.Context) {
	level, err := log.ParseLevel(loglevel)
	check(err)
	log.SetLevel(level)
	if bam == "" {
		log.Fatal("no file specified")
	}
	// stats := bamstats.Coverage1(bam, annotation, cpu)
	stats := bamstats.General(bam, cpu, maxBuf)
	bamstats.OutputJson(stats)
}

func main() {
	app := cli.NewApp()
	app.Name = "bamstats"
	app.Usage = "Compute mapping statistics"
	app.Version = bamstats.Version
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "bam, b",
			Value:       "",
			Usage:       "input file",
			Destination: &bam,
		},
		cli.StringFlag{
			Name:        "annotation, a",
			Value:       "",
			Usage:       "bgzip compressed and indexed annotation file",
			Destination: &annotation,
		},
		cli.StringFlag{
			Name:        "loglevel",
			Value:       "warn",
			Usage:       "logging level",
			Destination: &loglevel,
		},
		cli.IntFlag{
			Name:        "cpu, c",
			Value:       1,
			Usage:       "number of cpus to be used",
			Destination: &cpu,
		},
		cli.IntFlag{
			Name:        "max-buf",
			Value:       100000,
			Usage:       "maximum buffer size for reading",
			Destination: &maxBuf,
		},
	}
	app.Action = run

	if len(os.Args) == 1 {
		os.Args = append(os.Args, "help")
	}
	app.Run(os.Args)
}
