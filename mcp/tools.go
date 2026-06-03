package mcp

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerTools(server *mcp.Server, deps *Deps) {
	registerWhoamiTools(server, deps)
	registerS3Tools(server, deps)
	registerSQSTools(server, deps)
	registerSecretsTools(server, deps)
	registerSNSTools(server, deps)
	registerMSKTools(server, deps)
	registerCloudFrontTools(server, deps)
}
