// Binary redpolitika-pages implements the static.pages extension point.
// It serves .md files from a configurable directory via PagesService gRPC.
package main

import (
	"context"
	"flag"

	goplugin "github.com/hashicorp/go-plugin"
	identitypb "github.com/drupaldoesnotexists/redpolitika/ce/plugins/pages/proto/identity"
	pagespb "github.com/drupaldoesnotexists/redpolitika/ce/plugins/pages/proto/pages"
	"google.golang.org/grpc"
)

// pagesDir is the directory containing .md files to serve.
var pagesDir string

func main() {
	flag.StringVar(&pagesDir, "pages-dir", "./pages", "directory with .md files")
	flag.Parse()

	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: goplugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "REDPOLITIKA_PLUGIN",
			MagicCookieValue: "ce_v1",
		},
		Plugins: goplugin.PluginSet{
			"redpolitika-pages": &pagesGRPCPlugin{},
		},
		GRPCServer: goplugin.DefaultGRPCServer,
	})
}

// pagesGRPCPlugin satisfies both goplugin.Plugin and goplugin.GRPCPlugin.
// It registers PagesService and PluginIdentityService on the shared gRPC server.
type pagesGRPCPlugin struct {
	goplugin.NetRPCUnsupportedPlugin
}

func (p *pagesGRPCPlugin) GRPCServer(_ *goplugin.GRPCBroker, s *grpc.Server) error {
	pagespb.RegisterPagesServiceServer(s, &pagesService{})
	identitypb.RegisterPluginIdentityServer(s, &identityService{})
	return nil
}

func (p *pagesGRPCPlugin) GRPCClient(_ context.Context, _ *goplugin.GRPCBroker, conn *grpc.ClientConn) (interface{}, error) {
	return conn, nil
}
