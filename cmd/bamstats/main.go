package main

import (
	"fmt"
	"runtime"

	"github.com/guigolab/bamstats"
	"github.com/guigolab/bamstats/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

var (
	bam, annotation, loglevel, output string
	cpu, maxBuf, reads                int
	uniq                              bool
)

func run(cmd *cobra.Command, args []string) (err error) {
	err = nil

	// Set loglevel
	level, err := log.ParseLevel(loglevel)
	if err != nil {
		return
	}
	log.SetLevel(level)
	// Get stats
	logger := log.WithFields(log.Fields{
		"version":   version,
		"commit":    commit,
		"buildTime": date,
	})
	logger.Infof("Running %s", cmd.Use)
	log.Infof("Using %v out of %v logical CPUs", cpu, runtime.NumCPU())
	allStats, err := bamstats.Process(bam, annotation, cpu, maxBuf, reads, uniq)
	if err != nil {
		return
	}

	w := utils.NewWriter(output)
	allStats.OutputJSON(w)

	return
}

func setBamstatsFlags(c *cobra.Command) {
	c.PersistentFlags().StringVarP(&bam, "input", "i", "", "input file (required)")
	c.PersistentFlags().StringVarP(&annotation, "annotaion", "a", "", "element annotation file")
	c.PersistentFlags().StringVarP(&loglevel, "loglevel", "", "warn", "logging level")
	c.PersistentFlags().StringVarP(&output, "output", "o", "-", "output file")
	c.PersistentFlags().IntVarP(&cpu, "cpu", "c", runtime.NumCPU(), "number of cpus to be used")
	c.PersistentFlags().IntVarP(&maxBuf, "max-buf", "", 1000000, "maximum number of buffered records")
	c.PersistentFlags().IntVarP(&reads, "reads", "n", -1, "number of records to process")
	c.PersistentFlags().BoolVarP(&uniq, "uniq", "u", false, "output genomic coverage statistics for uniqely mapped reads too")
	// c.PersistentFlags().Bool("version", false, "show version and exit")
	c.MarkPersistentFlagRequired("input")

	c.SetVersionTemplate(`{{with .Name}}{{printf "== %s ==\n" .}}{{end}}{{printf "%s\n" .Version}}`)
}

func buildVersion(version, commit, date string) string {
	var result = fmt.Sprintf("version: %s", version)
	if commit != "" {
		result = fmt.Sprintf("%s\ncommit: %s", result, commit)
	}
	if date != "" {
		result = fmt.Sprintf("%s\nbuilt at: %s", result, date)
	}
	return result
}

func main() {
	var rootCmd = &cobra.Command{
		Use:     "bamstats",
		Short:   "Mapping statistics",
		Long:    "bamstats - compute mapping statistics",
		RunE:    run,
		Version: buildVersion(version, commit, date),
	}

	setBamstatsFlags(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Debug(err)
	}
}
