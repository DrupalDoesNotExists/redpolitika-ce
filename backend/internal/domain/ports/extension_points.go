package ports

import (
	"context"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"
)

// LLMProvider is an extension point for LLM-based detection.
// Implementation: plugin via HashiCorp go-plugin (or direct provider in EE).
type LLMProvider interface {
	// CheckText sends text for LLM analysis and returns matches.
	CheckText(ctx context.Context, text string, rule *model.Rule) ([]*model.Flag, error)
}

// DetectFunctionProvider is a generic extension point for custom detection (A27).
// Plugins register with a scoped name (A37, e.g., "spacy/ner") and are dispatched
// by the rule's detect.method. The domain does not know plugin capabilities in advance.
type DetectFunctionProvider interface {
	// Detect runs custom detection on text and returns flags.
	Detect(ctx context.Context, text string, rule *model.Rule) ([]*model.Flag, error)
}

// FixFunctionProvider is a generic extension point for custom fix functions (A27).
type FixFunctionProvider interface {
	// Fix applies a custom fix and returns the corrected text.
	// config is the JSON-serialized YAML args from the fix node (may be empty).
	// methodName is the plugin method name that was registered.
	Fix(ctx context.Context, text string, flag *model.Flag, config string, methodName string) (string, error)
}

// Migrator — A13 hybrid migrations.
type Migrator interface {
	Migrate(ctx context.Context, req MigrateRequest) (MigrateResult, error)
}

// MigrateRequest describes a migration run.
type MigrateRequest struct {
	Dialect       string
	DSN           string
	TargetVersion int64
	Direction     string
}

// MigrateResult reports migration outcome.
type MigrateResult struct {
	CurrentVersion int64
	ErrMsg         string
}

// FrontendBundleProvider — A17.
type FrontendBundleProvider interface {
	ListBundles(ctx context.Context) ([]model.FrontendBundle, error)
	GetAsset(ctx context.Context, plugin, path string) (data []byte, contentType string, err error)
}

// BrandingProvider — A36.
type BrandingProvider interface {
	Branding(ctx context.Context) (model.Branding, error)
}

// StorageProvider — DB access for plugins.
type StorageProvider interface {
	Dialect(ctx context.Context) (string, error)
	DSN(ctx context.Context) (string, error)
}

// AuthProvider authenticates callers.
type AuthProvider interface {
	Authenticate(ctx context.Context, token string) (subject string, err error)
}

// RolePermissionProvider checks authorization.
type RolePermissionProvider interface {
	Allowed(ctx context.Context, subject, action, resource string) (bool, error)
}

// LicenseProvider (BillingProvider/LicenseProvider A27).
type LicenseProvider interface {
	Check(ctx context.Context) (ok bool, reason string, err error)
}

// DocumentStorageBackend stores document blobs.
type DocumentStorageBackend interface {
	Get(ctx context.Context, id string) ([]byte, error)
	Put(ctx context.Context, id string, data []byte) error
	Delete(ctx context.Context, id string) error
}

// AuditLogger records audit events.
type AuditLogger interface {
	Log(ctx context.Context, event string, fields map[string]string) error
}

// MetricsExporter exports metrics.
type MetricsExporter interface {
	Export(ctx context.Context, name string, value float64, labels map[string]string) error
}

// WebhookProvider delivers outbound webhooks.
type WebhookProvider interface {
	Deliver(ctx context.Context, event string, payload []byte) error
}

// StaticPagesProvider — static content pages from plugins (Level 1 data EP).
type StaticPagesProvider interface {
	ListPages(ctx context.Context) ([]model.Page, error)
	GetPage(ctx context.Context, slug string) (*model.Page, error)
}

// RuleValidator — additional plugin validation on load (core RE2 stays in NewRule).
type RuleValidator interface {
	ValidateRule(ctx context.Context, rule *model.Rule) error
}
