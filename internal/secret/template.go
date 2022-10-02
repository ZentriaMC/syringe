package secret

import (
	"context"
	"io"
	"text/template"

	cctx "github.com/ZentriaMC/syringe/internal/ctx"
	"github.com/ZentriaMC/syringe/internal/templatemap"
)

func Render(ctx context.Context, w io.Writer) (err error) {
	unit, credential := cctx.CredentialRequest(ctx)
	tm := cctx.TemplateMap(ctx)

	// Get template
	var credTmpl templatemap.CredentialTemplate
	if credTmpl, err = tm.GetTemplate(unit, credential); err != nil {
		return
	}

	opts := &TemplateOptions{
		CredentialTemplate: credTmpl,
		UnitName:           unit,
		CredentialName:     credential,
		FunctionsBlacklist: []string{
			//"sprig_env",
			//"sprig_expandenv",
		},
	}

	tmpl := template.New("")
	tmpl.Delims(credTmpl.DelimLeft, credTmpl.DelimRight)
	tmpl.Funcs(funcMap(ctx, opts))
	if credTmpl.AllowMissing {
		tmpl.Option("missingkey=zero")
	} else {
		tmpl.Option("missingkey=error")
	}

	tmpl, err = tmpl.Parse(credTmpl.Content)
	if err != nil {
		return
	}

	writer := NewHardLimitWriter(w, MAX_CREDENTIAL_SIZE)
	err = tmpl.Execute(writer, nil)
	return
}
