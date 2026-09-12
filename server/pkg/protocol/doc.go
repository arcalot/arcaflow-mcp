// Package protocol implements MCP JSON-RPC handling and method routing.
//
// Supported MCP methods:
// - initialize / initialized
// - tools/list, tools/call
// - resources/list, resources/read
// - ping
//
// The handler enforces JSON-RPC 2.0 validation rules and returns MCP-compliant
// error responses for invalid requests or parameters.
package protocol
