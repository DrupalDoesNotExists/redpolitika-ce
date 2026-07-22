package plugin

// Known extension point capability IDs (A27).
// Each maps to a gRPC service contract the core knows at compile time.
// Plugins declare which they implement via GetCapabilities; the core
// uses this registry to construct typed gRPC clients.
const (
	CapPages          = "static.pages"    // pages.PagesService — Level 1 data EP (A27)
	CapDetectProvider = "detect.provider" // detcet.DetectService
	CapLLMProvider    = "llm.provider"    // llm.LLMService
	CapFixProvider    = "fix.provider"    // fix.FixService
	CapMigrator       = "migrator.run"    // migrator.MigratorService

	// EE-only extension points (declared for documentation, no proto yet):
	CapFrontendBundle = "frontend.bundle" // FrontendBundleProvider
	CapBranding       = "branding.provide"
	CapStorage        = "storage.dialect"
	CapAuth           = "auth.authenticate"
	CapAuthz          = "auth.authorize"
	CapLicense        = "license.check"
	CapDocumentStore  = "document.store"
	CapAuditLog       = "audit.log"
	CapMetrics        = "metrics.export"
	CapWebhook        = "webhook.deliver"
	CapRuleValidator  = "rule.validate"
)

// KnownCapabilities maps capability IDs to their human-readable gRPC service names.
// Used for introspection / error messages.
var KnownCapabilities = map[string]string{
	CapPages:          "PagesService",
	CapDetectProvider: "DetectService",
	CapLLMProvider:    "LLMService",
	CapFixProvider:    "FixService",
	CapMigrator:       "MigratorService",

	CapFrontendBundle: "FrontendBundleProvider",
	CapBranding:       "BrandingProvider",
	CapStorage:        "StorageProvider",
	CapAuth:           "AuthProvider",
	CapAuthz:          "RolePermissionProvider",
	CapLicense:        "LicenseProvider",
	CapDocumentStore:  "DocumentStorageBackend",
	CapAuditLog:       "AuditLogger",
	CapMetrics:        "MetricsExporter",
	CapWebhook:        "WebhookProvider",
	CapRuleValidator:  "RuleValidator",
}

// BuiltinMethods is removed — all plugin methods are scoped and registered
// at plugin load time via GetCapabilities. No method names are hardcoded in core.
