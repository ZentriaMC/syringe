package secret

import (
	"context"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

func funcMap(ctx context.Context, opts *TemplateOptions) (fm template.FuncMap) {
	t := &templateFn{
		ctx:          ctx,
		sandboxPath:  opts.SandboxPath,
		cachedValues: make(map[string]any),
	}

	fm = template.FuncMap{
		"unitname":       t.unitName,
		"credentialname": t.credentialName,
		"file":           t.readFile,
		"vault_read":     t.readVault,

		"b64decode": b64decode,
		"b64encode": b64encode,
		"sha256sum": sha256sum,
		"sha1sum":   sha1sum,
		"md5sum":    md5sum,
		"time":      timef,

		"sockaddr":  sockaddr,
		"spew_dump": spewDump,
	}

	for name, fn := range sprig.FuncMap() {
		name := "sprig_" + name
		fn := fn

		fm[name] = fn
	}

	for _, name := range opts.FunctionsBlacklist {
		if _, ok := fm[name]; ok {
			fm[name] = disabledFunc(name)
		}
	}

	return
}

func disabledFunc(name string) func(...any) (any, error) {
	return func(...any) (any, error) {
		return nil, fmt.Errorf("function '%s' is disabled", name)
	}
}
