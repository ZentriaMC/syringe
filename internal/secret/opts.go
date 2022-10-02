package secret

import "github.com/ZentriaMC/syringe/internal/templatemap"

type TemplateOptions struct {
	templatemap.CredentialTemplate
	UnitName           string
	CredentialName     string
	FunctionsBlacklist []string
}
