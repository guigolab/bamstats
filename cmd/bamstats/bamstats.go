package main

import (
	"runtime"

	log "github.com/Sirupsen/logrus"
	"github.com/guigolab/bamstats"
	"github.com/spf13/cobra"
)

var (
	bam, annotation, loglevel, output string
	cpu, maxBuf, reads                int
	uniq                              bool
)

func run(cmd *cobra.Command, args []string) (err error) {
	err = nil
	if hasVersionFlag(cmd) {
		return
	}

	// Set loglevel
	level, err := log.ParseLevel(loglevel)
	if err != nil {
		return
	}
	log.SetLevel(level)
	// Get stats
	log.Infof("Running %s %s", cmd.Use, bamstats.Version())
	log.Infof("Using %v out of %v logical CPUs", cpu, runtime.NumCPU())
	stats, err := bamstats.Process(bam, annotation, cpu, maxBuf, reads, uniq)
	if err != nil {
		return
	}

	bamstats.WriteOutput(output, stats)

	return
}

func setBamstatsFlags(c *cobra.Command) {
	c.PersistentFlags().StringVarP(&bam, "input", "i", "", "input file")
	c.PersistentFlags().StringVarP(&annotation, "annotaion", "a", "", "element annotation file")
	c.PersistentFlags().StringVarP(&loglevel, "loglevel", "", "warn", "logging level")
	c.PersistentFlags().StringVarP(&output, "output", "o", "-", "output file")
	c.PersistentFlags().IntVarP(&cpu, "cpu", "c", runtime.NumCPU(), "number of cpus to be used")
	c.PersistentFlags().IntVarP(&maxBuf, "max-buf", "", 1000000, "maximum number of buffered records")
	c.PersistentFlags().IntVarP(&reads, "reads", "n", -1, "number of records to process")
	c.PersistentFlags().BoolVarP(&uniq, "uniq", "u", false, "output genomic coverage statistics for uniqely mapped reads too")
	c.PersistentFlags().Bool("version", false, "show version and exit")
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "bamstats",
		Short: "Mapping statistics",
		Long:  "bamstats - compute mapping statistics",
		RunE:  run,
	}

	setBamstatsFlags(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Debug(err)
	}
}
