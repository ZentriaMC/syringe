package ctx

import (
	"context"

	"github.com/ZentriaMC/syringe/internal/templatemap"
	vaultapi "github.com/hashicorp/vault/api"
)

type ContextValueFunc func(ctx context.Context) context.Context

func Apply(ctx context.Context, values ...ContextValueFunc) context.Context {
	for _, fn := range values {
		ctx = fn(ctx)
	}
	return ctx
}

func WithTemplateMap(tm *templatemap.TemplateMap) ContextValueFunc {
	return func(ctx context.Context) context.Context {
		ctx = context.WithValue(ctx, templateMap, tm)
		return ctx
	}
}

func TemplateMap(ctx context.Context) *templatemap.TemplateMap {
	return ctx.Value(templateMap).(*templatemap.TemplateMap)
}

func WithCredentialRequest(unit string, credential string) ContextValueFunc {
	return func(ctx context.Context) context.Context {
		ctx = context.WithValue(ctx, credentialReqUnit, unit)
		ctx = context.WithValue(ctx, credentialReqCred, credential)
		return ctx
	}
}

func CredentialRequest(ctx context.Context) (unit, secret string) {
	unit = ctx.Value(credentialReqUnit).(string)
	secret = ctx.Value(credentialReqCred).(string)
	return
}

func WithVaultClient(client *vaultapi.Client) ContextValueFunc {
	return func(ctx context.Context) context.Context {
		return context.WithValue(ctx, vaultClient, client)
	}
}

func VaultClient(ctx context.Context) *vaultapi.Client {
	v := ctx.Value(vaultClient)
	if v == nil {
		return nil
	}

	if v, ok := v.(*vaultapi.Client); ok {
		return v
	}

	return nil
}

func WithSocketPaths(sp []string) ContextValueFunc {
	return func(ctx context.Context) context.Context {
		return context.WithValue(ctx, socketPath, sp)
	}
}

func SocketPaths(ctx context.Context) (v []string) {
	v = ctx.Value(socketPath).([]string)
	return
}

func WithGlobalDebug(gd bool) ContextValueFunc {
	return func(ctx context.Context) context.Context {
		return context.WithValue(ctx, globalDebug, gd)
	}
}

func GlobalDebug(ctx context.Context) (v bool) {
	v = ctx.Value(globalDebug).(bool)
	return
}
