// Package mcp implements a Model Context Protocol server for aws-probe.
//
// Use [Register] to attach tools, resources, and prompts to an existing MCP server
// (for example kubectl-netdrill with --external-tools). Use [Run] for a
// standalone stdio server via the aws-probe mcp command.
package mcp
