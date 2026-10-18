package termforge

// Graph is plain data: a title and one series with a label per point (for
// example dates). GraphWidget does all the painting.
type Graph struct {
	title  string
	labels []string
	values []float64
}

func NewGraph() *Graph { return &Graph{} }

func (g *Graph) SetTitle(s string) { g.title = s }
func (g *Graph) Title() string     { return g.title }

// SetPoints replaces the series. labels may be shorter than values; a missing
// label is shown empty.
func (g *Graph) SetPoints(labels []string, values []float64) {
	g.labels = append([]string(nil), labels...)
	g.values = append([]float64(nil), values...)
}

func (g *Graph) Len() int { return len(g.values) }

func (g *Graph) label(i int) string {
	if i < 0 || i >= len(g.labels) {
		return ""
	}
	return g.labels[i]
}
