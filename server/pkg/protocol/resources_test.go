package protocol

import (
	"context"
	"encoding/json"
	"testing"
)

type stubResourceProvider struct {
	items    []ResourceItem
	contents map[string]ResourceContent
}

func (provider stubResourceProvider) List(
	_ context.Context,
) ([]ResourceItem, *ErrorObject) {
	return provider.items, nil
}

func (provider stubResourceProvider) Read(
	_ context.Context,
	uri string,
) (*ResourceContent, bool, *ErrorObject) {
	content, ok := provider.contents[uri]
	if !ok {
		return nil, false, nil
	}
	return &content, true, nil
}

func TestResourcesListReturnsProviders(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})
	server.RegisterResourceProvider(stubResourceProvider{
		items: []ResourceItem{
			{
				URI:      "workflow-schema://filesystem?location=/workflows&path=one.yaml",
				Name:     "workflow-one",
				MimeType: "application/json",
			},
		},
		contents: map[string]ResourceContent{},
	})

	initReq := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(1),
		Method:  "initialize",
		Params:  mustMarshalRaw(InitializeParams{ProtocolVersion: ProtocolVersion}),
	})
	if _, err := server.Handle(context.Background(), initReq); err != nil {
		t.Fatalf("initialize failed: %v", err)
	}
	initialized := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		Method:  "initialized",
	})
	if _, err := server.Handle(context.Background(), initialized); err != nil {
		t.Fatalf("initialized failed: %v", err)
	}

	listReq := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(2),
		Method:  "resources/list",
	})
	responses, err := server.Handle(context.Background(), listReq)
	if err != nil {
		t.Fatalf("resources/list failed: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}

	var resp Response
	if err := json.Unmarshal(responses[0], &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %#v", resp.Error)
	}
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected result map")
	}
	resources, _ := result["resources"].([]interface{})
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
}

func TestResourcesReadReturnsContent(t *testing.T) {
	server := NewServer(nil, ServerInfo{Name: "arcaflow-mcp"})
	server.RegisterResourceProvider(stubResourceProvider{
		items: []ResourceItem{},
		contents: map[string]ResourceContent{
			"workflow-example://filesystem?location=/workflows&path=one.yaml": {
				URI:      "workflow-example://filesystem?location=/workflows&path=one.yaml",
				MimeType: "application/json",
				Text:     `{"example_input":{"value":"ok"}}`,
			},
		},
	})

	initReq := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(1),
		Method:  "initialize",
		Params:  mustMarshalRaw(InitializeParams{ProtocolVersion: ProtocolVersion}),
	})
	if _, err := server.Handle(context.Background(), initReq); err != nil {
		t.Fatalf("initialize failed: %v", err)
	}
	initialized := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		Method:  "initialized",
	})
	if _, err := server.Handle(context.Background(), initialized); err != nil {
		t.Fatalf("initialized failed: %v", err)
	}

	readReq := mustMarshal(Request{
		JSONRPC: JSONRPCVersion,
		ID:      rawID(2),
		Method:  "resources/read",
		Params: mustMarshalRaw(ResourcesReadParams{
			URI: "workflow-example://filesystem?location=/workflows&path=one.yaml",
		}),
	})
	responses, err := server.Handle(context.Background(), readReq)
	if err != nil {
		t.Fatalf("resources/read failed: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}

	var resp Response
	if err := json.Unmarshal(responses[0], &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %#v", resp.Error)
	}
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected result map")
	}
	contents, _ := result["contents"].([]interface{})
	if len(contents) != 1 {
		t.Fatalf("expected 1 content entry, got %d", len(contents))
	}
}
