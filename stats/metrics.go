package stats

import (
	"fmt"
	"html/template"
	"io"
)

// Metrics defines an interface for a metric
type Metrics interface {
	Calculate(m Map) error
}

type metric float64

// IHECmetrics represents statistics for mapped reads
type IHECmetrics struct {
	Mapped     metric `json:"FRACTION_MAPPED,omitempty"`
	Intergenic metric `json:"FRACTION_INTERGENIC,omitempty"`
	RRNA       metric `json:"FRACTION_RRNA,omitempty"`
	Duplicated metric `json:"FRACTION_DUPLICATED,omitempty"`
}

// Calculate compute metrics from general stats
func (m *IHECmetrics) Calculate(sm Map) error {
	g, hasGeneral := sm["general"].(*GeneralStats)
	c, hasCoverage := sm["ihec"].(*IHECstats)

	if hasGeneral {
		mapped := metric(g.Reads.Mapped.Total())
		m.Mapped = mapped / metric(g.Reads.Total)
		if hasCoverage {
			m.Intergenic = metric(c.Intergenic) / mapped
			m.RRNA = metric(c.RRNA) / mapped
			m.Duplicated = metric(g.Reads.Duplicated) / mapped
		}
	}

	return nil
}

// Output write metrics to out
func (m *IHECmetrics) Output(out io.Writer) {
	tmpl := `FRACTION_MAPPED	{{.Mapped}}
FRACTION_INTERGENIC	{{.Intergenic}}
FRACTION_RRNA	{{.RRNA}}
FRACTION_DUPLICATED	{{.Duplicated}}
`
	o := template.Must(template.New("IHEC").Parse(tmpl))
	o.Execute(out, m)

}

func (m metric) String() string {
	return fmt.Sprintf("%.6g", float64(m))
}

// if string(element) == "intergenic" {
// 	if feat.End()-feat.Start() > 1000 {
// 		feat, err = parseFeature([]byte(f.Chr()), element, feat.Start()+500, feat.End()-500, f.tags)
// 	} else {
// 		feat, err = nil, nil
// 	}
// }
