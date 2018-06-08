# exons
awk '$3 == "exon"' ${annotation} | sort -k1,1 -k4,4n | mergeBed -i stdin | awk -v OFS="\t" '$(NF+1)="exon"' > exons
# genes
awk '$3 == "gene"' ${annotation} | sort -k1,1 -k4,4n | mergeBed -i stdin | awk -v OFS="\t" '$(NF+1)="gene"' > genes
# introns
subtractBed -a genes -b exons | awk -v OFS="\t" '$(NF)="intron"' > introns
# intergenic
complementBed -i genes -g <(sort -k1,1 ${genome}) | awk -v OFS="\t" '$(NF+1)="intergenic"' > intergenic
# all
cat exons genes introns intergenic | sort -k1,1 -k2,2n > ${annotation}.elements.bed
