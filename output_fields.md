# Bamstats Output Fields

This document describes the fields of the different sections of a Bamstats output file.

## General

The `general` section contains mapping statistics collected from the input `BAM` file.

### Fields

#### `protocol`

The protocol used for sequencing, extracted from the SAM flags. It can be either `SingleEnd` or `PairedEnd`.

#### `reads`

Statistics and counts for single reads. In case of `PairedEnd` data each mate is counted independently.

##### `total`

The total number of reads, in `samtools view -c -F256`. It corresponds to the sum of [unmapped reads](#unmapped)  and all the values in the [mapped reads](#mapped) object.

##### `unmapped`

The number of unmapped reads, as in `samtools view -c -f4`.

##### `mapped`

This is an object containing the number of mapped reads grouped by the number of hits each read has (`NH` tag in the `SAM` format). The sum of these values gives the total number of mapped reads.

##### `mappings`

An object containing the following information on the alignments:

- the global `ratio` of the total number of mappings over the total number of mapped reads, representing the average number of hits per read
- the total number of mappings as in `samtools view -c`, including muliple hits for each read

#### `pairs`

Statistics and counts for read pairs, additionally reported if the data proocol is `PairedEnd`. The same fields as in the [reads](#reads) section, except for the [mappings](#mappings) object, are included but referring to read pairs.

### Fields

##### `insert_sizes`

An object containing the count of mapped pairs grouped by the corresponding insert size length.

## Genomic Coverage

The `genomeCoverage` section contains metrics for genomic coverage based on the provided annotation. The counts are computed for `continuous` and `split` reads. For `split` reads the aligned blocks are considered separately. An aggregated report with the `total` counts is also collected.

An additional genomic coverage section for uniquely mapped reads called `genomeCoverageUniq` is additionally reported in the output file when the `--uniq` (or `-u`) command line option is used.

### Fields

#### `exon`

Reads mapping to an exonic region. Reads must be totally included. For `split` reads, all the blocks must be included in an exonic region.

#### `intron`

Reads mapping to an intronic region. Reads must be totally included. For `split` reads, all the blocks must be included in an intronic region.

#### `exonic_intronic`

Reads overlapping  exon-intron junction. For `split` reads, any of the blocks can either overlap the junction, map to an `exon` or an `intron`.

#### `intergenic`

Reads mapping to an intergenic region. Reads must be totally included. For `split` reads, all the blocks must be included in an intergenic region.

#### `others`

Reads mapping to regions different from the ones described above. For `split` reads, this can also reported for unexpected regions combination of the alignment blocks.

## RNAseq

The `rnaseq` sections contains metrics computed following the recommendations from the IHEC Assay Standards working group.

### Fields

#### `intergenic`

The number of reads mapping to intergenic regions. Reads don't need to be totally included into the region and the whole read is used in case of `split` reads, so the metric is different from the [intergenic field from the genomic coverage](#intergenic) section.

#### `rRNA`

Number of aligned reads mapping to Ribosomal RNA regions. Regions are extracted from the provided annotation using the following values of the `gene_type` attribute:

- `rRNA`
- `Mt_rRNA`

#### `metrics`

Fractional metrics for the following read types:

|              |                                                                               |
|-------------:|-------------------------------------------------------------------------------|
|     `mapped` | mapped reads over the total number of reads                                   |
| `intergenic` | number of reads falling in intergenic regions over the number of mapped reads |
|       `rRNA` | number of reads falling in ribosomal regions over the number of mapped reads  |
| `duplicates` | number of duplicate reads over the number of mapped reads                     |
