# Bamstats

[![Build Status](https://travis-ci.org/guigolab/bamstats.svg?branch=master)](https://travis-ci.org/guigolab/bamstats)
[![Coverage Status](https://coveralls.io/repos/github/guigolab/bamstats/badge.svg?branch=master)](https://coveralls.io/github/guigolab/bamstats)

`Bamstats` is a command line tool written in `Go` for computing mapping statistics from a `BAM` file.

It currently supports the following statistics:

- general
- genome coverage

## General statistics

The general mapping statistics include:

- Total number of reads
- Number of unmapped reads
- Number of mapped reads grouped by number of multimaps (`NH` tag in `BAM` file)
- Number of mappings
- Ratio of mappings vs mapped reads

If the data is paired-end, a section for read-pairs is also reported. In addition to the above metrics, the section contains a map of the insert size length and the corresponding support as number of reads.

## Genome coverage statistics

The genome coverage ststistics are computed for RNA-seq data and include counts for the following genomic regions:

- exon
- intron
- exonic_intronic
- intergenic
- others

The above metrics are computed for continuous and split mapped reads. An aggregated total is computed across elements and read types too.

The `--uniq` (or `-u`) command line flag allows reporting of genome coverage statistics for uniquely mapped reads too.

## Output examples:

Some examples of the program output can be found in the `data` folder within the GitHub repository:

- link:data/expected-general.json[General stats]
- link:data/expected-coverage.json[Genomic coverage stats]
- link:data/expected-coverage-uniq.json[Genomic coverage stats with uniquely mapped reads] (Note that the `coverageUniq` stats are reported as an additional JSON object)

## License

This software is release under a BSD-style license that can be found in the `LICENSE` file.