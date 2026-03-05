package report

import (
	"io"
	"strings"
	"text/template"

	"github.com/dshills/verifier/internal/domain"
)

var mdFuncs = template.FuncMap{
	"severities": func() []string {
		return []string{"critical", "high", "medium", "low"}
	},
	"title": func(s string) string {
		if len(s) == 0 {
			return s
		}
		return strings.ToUpper(s[:1]) + s[1:]
	},
	"truncate": func(s string, n int) string {
		if len(s) <= n {
			return s
		}
		return s[:n] + "..."
	},
	"code": func(s string) string {
		return "`" + s + "`"
	},
}

// WriteMarkdown writes the report as markdown.
func WriteMarkdown(w io.Writer, rpt *domain.Report) error {
	tmpl := template.Must(template.New("md").Funcs(mdFuncs).Parse(mdTmpl))
	return tmpl.Execute(w, rpt)
}

//nolint:lll
var mdTmpl = "# Verifier Report\n\n" +
	"**Tool:** {{ .Meta.Tool }} v{{ .Meta.Version }}  \n" +
	"**Repository:** {{ .Meta.RepoRoot }}  \n" +
	"**Mode:** {{ .Meta.Mode }}  \n" +
	"**Timestamp:** {{ .Meta.Timestamp }}\n\n" +
	"## Summary\n\n" +
	"| Metric | Value |\n|--------|-------|\n" +
	"| Risk Score | {{ .Summary.RiskScore }} |\n" +
	"| Total Findings | {{ .Summary.TotalFindings }} |\n" +
	"| Truncated | {{ .Summary.Truncated }} |\n" +
	"| Missing Recommendations | {{ .Summary.MissingRecommendations }} |\n" +
	"| Unverifiable Requirements | {{ .Summary.UnverifiableRequirements }} |\n" +
	"{{ range $sev := severities }}\n## {{ title $sev }} Gaps\n" +
	"{{ range $.Recommendations }}{{ if eq .Severity $sev }}\n" +
	"### {{ .ID }}: {{ .Proposal.Title }}\n\n" +
	"- **Category:** {{ .Category }}\n" +
	"- **Severity:** {{ .Severity }}\n" +
	"- **Confidence:** {{ .Confidence }}\n" +
	"- **Target:** {{ .Target.Kind }} {{ code .Target.Name }} ({{ .Target.File }})\n" +
	"- **Approach:** {{ .Proposal.Approach }}\n" +
	"{{ if .Proposal.Assertions }}\n**Assertions:**\n{{ range .Proposal.Assertions }}- {{ . }}\n{{ end }}{{ end }}" +
	"{{ if .ExistingTests }}\n**Existing Tests:**\n{{ range .ExistingTests }}- {{ .Name }} ({{ .File }}){{ if .Gap }} -- gap: {{ .Gap }}{{ end }}\n{{ end }}{{ end }}" +
	"{{ end }}{{ end }}{{ end }}" +
	"{{ if .Requirements }}\n## Requirements\n\n" +
	"| ID | Verifiability | Text |\n|----|--------------|------|\n" +
	"{{ range .Requirements }}| {{ .ID }} | {{ .Verifiability }} | {{ truncate .Text 80 }} |\n{{ end }}{{ end }}\n"
