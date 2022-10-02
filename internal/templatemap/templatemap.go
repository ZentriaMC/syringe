package templatemap

import (
	"fmt"
	"sync"

	"github.com/ZentriaMC/syringe/internal/config"
	"go.uber.org/multierr"
)

type CredentialTemplate struct {
	DelimLeft    string
	DelimRight   string
	AllowMissing bool
	SandboxPath  string

	Content string
}

type TemplateMap struct {
	lock sync.RWMutex

	// unit -> template
	catchall map[string]CredentialTemplate
	// unit -> credential -> template
	templates map[string]map[string]CredentialTemplate
}

func NewTemplateMap() *TemplateMap {
	return &TemplateMap{
		catchall:  make(map[string]CredentialTemplate),
		templates: make(map[string]map[string]CredentialTemplate),
	}
}

func (tm *TemplateMap) Populate(config *config.Config) (err error) {
	newCatchall := make(map[string]CredentialTemplate)
	newTemplates := make(map[string]map[string]CredentialTemplate)

	for _, template := range config.Templates {
		unit := template.Unit
		catchAll := len(template.Credential) == 0

		var lerr error
		var ct CredentialTemplate
		if ct, lerr = toCredentialTemplate(template); lerr != nil {
			err = multierr.Append(err, lerr)
			continue
		}

		if catchAll {
			_, ok := newCatchall[unit]
			if ok {
				err = multierr.Append(err, fmt.Errorf("catch-all template for unit '%s' is already set", unit))
				continue
			}

			newCatchall[unit] = ct
		} else {

			for _, credential := range template.Credential {
				credTemplates, ok := newTemplates[unit]
				if !ok {
					credTemplates = make(map[string]CredentialTemplate)
				}

				_, ok = credTemplates[credential]
				if ok {
					err = multierr.Append(err, fmt.Errorf("template for unit '%s' credential '%s' is already set", unit, credential))
					continue
				}

				credTemplates[credential] = ct
				newTemplates[unit] = credTemplates
			}
		}
	}
	if err != nil {
		return
	}

	tm.lock.Lock()
	defer tm.lock.Unlock()

	tm.catchall = newCatchall
	tm.templates = newTemplates
	return
}

func toCredentialTemplate(template config.Template) (t CredentialTemplate, err error) {
	if opts := template.Options; opts != nil {
		t.DelimLeft = opts.DelimLeft
		t.DelimRight = opts.DelimRight
		t.AllowMissing = opts.AllowMissing
		t.SandboxPath = opts.SandboxPath
	} else {
		t.DelimLeft = "{{"
		t.DelimRight = "}}"
	}
	t.Content = template.Contents
	return
}

func (tm *TemplateMap) GetTemplate(unit string, credential string) (tmpl CredentialTemplate, err error) {
	tm.lock.RLock()
	defer tm.lock.RUnlock()

	var ok bool

	// Try specific secret template first
	if credentialTemplates, ok := tm.templates[unit]; ok {
		if tmpl, ok = credentialTemplates[credential]; ok {
			return
		}
	}

	// Try catch-all template
	if tmpl, ok = tm.catchall[unit]; ok {
		return
	}

	err = fmt.Errorf("no template for unit=%s credential=%s", unit, credential)
	return
}
