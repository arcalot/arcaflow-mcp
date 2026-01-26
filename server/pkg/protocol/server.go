package protocol

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"sync"
)

type connectionState int

const (
	stateNew connectionState = iota
	stateInitialized
	stateReady
)

// ToolHandler executes a registered MCP tool.
type ToolHandler func(
	ctx context.Context,
	arguments map[string]interface{},
) (ToolsCallResult, *ErrorObject)

// ToolRegistration binds a tool definition to its handler.
type ToolRegistration struct {
	Definition ToolDefinition
	Handler    ToolHandler
}

// Server routes MCP requests to protocol handlers.
type Server struct {
	logger     *slog.Logger
	serverInfo ServerInfo
	state      connectionState
	stateMu    sync.Mutex
	tools      map[string]ToolRegistration
	resources  []ResourceProvider
}

// NewServer constructs an MCP protocol server with default capabilities.
func NewServer(logger *slog.Logger, serverInfo ServerInfo) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	if serverInfo.Name == "" {
		serverInfo.Name = "arcaflow-mcp"
	}
	return &Server{
		logger:     logger,
		serverInfo: serverInfo,
		state:      stateNew,
		tools:      make(map[string]ToolRegistration),
		resources:  []ResourceProvider{},
	}
}

// RegisterTool adds a tool to the server registry.
func (s *Server) RegisterTool(registration ToolRegistration) {
	if registration.Definition.Name == "" || registration.Handler == nil {
		return
	}
	s.tools[registration.Definition.Name] = registration
}

// RegisterResourceProvider adds a resource provider to the server registry.
func (s *Server) RegisterResourceProvider(provider ResourceProvider) {
	if provider == nil {
		return
	}
	s.resources = append(s.resources, provider)
}

// Handle processes a JSON-RPC message and returns serialized responses.
func (s *Server) Handle(
	ctx context.Context,
	payload []byte,
) ([][]byte, error) {
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) == 0 {
		return s.encodeResponses([]Response{*invalidRequestResponse(nil)})
	}

	if trimmed[0] == '[' {
		return s.handleBatch(ctx, trimmed)
	}

	return s.handleSingle(ctx, trimmed)
}

func (s *Server) handleSingle(
	ctx context.Context,
	payload []byte,
) ([][]byte, error) {
	var req Request
	if err := json.Unmarshal(payload, &req); err != nil {
		return s.encodeResponses([]Response{parseErrorResponse()})
	}

	resp := s.handleRequest(ctx, req)
	if resp == nil {
		return nil, nil
	}

	return s.encodeResponses([]Response{*resp})
}

func (s *Server) handleBatch(
	ctx context.Context,
	payload []byte,
) ([][]byte, error) {
	var requests []Request
	if err := json.Unmarshal(payload, &requests); err != nil {
		return s.encodeResponses([]Response{parseErrorResponse()})
	}

	if len(requests) == 0 {
		return s.encodeResponses([]Response{*invalidRequestResponse(nil)})
	}

	var responses []Response
	for _, req := range requests {
		resp := s.handleRequest(ctx, req)
		if resp != nil {
			responses = append(responses, *resp)
		}
	}

	if len(responses) == 0 {
		return nil, nil
	}

	return s.encodeResponses(responses)
}

func (s *Server) handleRequest(
	ctx context.Context,
	req Request,
) *Response {
	if !isValidRequest(req) {
		return invalidRequestResponse(req.ID)
	}

	if !req.IDSet {
		s.handleNotification(ctx, req)
		return nil
	}

	switch req.Method {
	case "initialize":
		return s.handleInitialize(ctx, req)
	case "initialized":
		return s.handleInitialized(req)
	case "tools/list":
		return s.handleToolsList(ctx, req)
	case "tools/call":
		return s.handleToolsCall(ctx, req)
	case "resources/list":
		return s.handleResourcesList(ctx, req)
	case "resources/read":
		return s.handleResourcesRead(ctx, req)
	case "ping":
		return s.handlePing(ctx, req)
	default:
		return methodNotFoundResponse(req.ID, req.Method)
	}
}

func (s *Server) handleInitialize(
	ctx context.Context,
	req Request,
) *Response {
	if len(req.Params) == 0 {
		return errorResponse(
			req.ID,
			newError(ErrInvalidParams, "initialize params required", nil),
		)
	}

	if _, errResp := requireObjectParams(req.ID, req.Params); errResp != nil {
		return errResp
	}

	var params InitializeParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errorResponse(
			req.ID,
			newError(ErrInvalidParams, "invalid initialize params", nil),
		)
	}

	if params.ProtocolVersion == "" {
		return errorResponse(
			req.ID,
			newError(ErrInvalidParams, "protocolVersion required", nil),
		)
	}

	if params.ProtocolVersion != ProtocolVersion {
		return errorResponse(
			req.ID,
			newError(
				ErrInvalidParams,
				"unsupported protocolVersion",
				map[string]string{"supported": ProtocolVersion},
			),
		)
	}

	if !s.transitionFrom(stateNew, stateInitialized) {
		return errorResponse(
			req.ID,
			newError(ErrInvalidRequest, "initialize already completed", nil),
		)
	}

	result := InitializeResult{
		ProtocolVersion: ProtocolVersion,
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{ListChanged: false},
			Resources: &ResourcesCapability{
				Subscribe:   false,
				ListChanged: false,
			},
		},
		ServerInfo: s.serverInfo,
	}

	return resultResponse(req.ID, result)
}

func (s *Server) handleInitialized(req Request) *Response {
	switch s.getState() {
	case stateInitialized:
		s.transitionFrom(stateInitialized, stateReady)
	case stateReady:
		// Repeated initialized requests are safe to ignore.
	default:
		s.logger.Warn("initialized received before initialize")
		return errorResponse(
			req.ID,
			newError(ErrInvalidRequest, "initialize required first", nil),
		)
	}

	return resultResponse(req.ID, map[string]interface{}{})
}

func (s *Server) handleToolsList(
	ctx context.Context,
	req Request,
) *Response {
	if errResp := s.requireReady(req.ID); errResp != nil {
		return errResp
	}

	var params ToolsListParams
	if len(req.Params) > 0 {
		if _, errResp := requireObjectParams(req.ID, req.Params); errResp != nil {
			return errResp
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return errorResponse(
				req.ID,
				newError(ErrInvalidParams, "invalid tools/list params", nil),
			)
		}
	}

	_ = ctx
	_ = params

	result := ToolsListResult{
		Tools: s.listTools(),
	}

	return resultResponse(req.ID, result)
}

func (s *Server) handleToolsCall(
	ctx context.Context,
	req Request,
) *Response {
	if errResp := s.requireReady(req.ID); errResp != nil {
		return errResp
	}

	if len(req.Params) == 0 {
		return errorResponse(
			req.ID,
			newError(ErrInvalidParams, "tools/call params required", nil),
		)
	}

	if _, errResp := requireObjectParams(req.ID, req.Params); errResp != nil {
		return errResp
	}

	var params ToolsCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errorResponse(
			req.ID,
			newError(ErrInvalidParams, "invalid tools/call params", nil),
		)
	}

	if params.Name == "" {
		return errorResponse(
			req.ID,
			newError(ErrInvalidParams, "tool name required", nil),
		)
	}

	registration, ok := s.tools[params.Name]
	if !ok {
		return errorResponse(
			req.ID,
			newError(ErrInvalidParams, "tool not found", nil),
		)
	}

	result, errObj := registration.Handler(ctx, params.Arguments)
	if errObj != nil {
		return errorResponse(req.ID, errObj)
	}

	return resultResponse(req.ID, result)
}

func (s *Server) handleResourcesList(
	ctx context.Context,
	req Request,
) *Response {
	if errResp := s.requireReady(req.ID); errResp != nil {
		return errResp
	}

	var params ResourcesListParams
	if len(req.Params) > 0 {
		if _, errResp := requireObjectParams(req.ID, req.Params); errResp != nil {
			return errResp
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return errorResponse(
				req.ID,
				newError(ErrInvalidParams, "invalid resources/list params", nil),
			)
		}
	}

	result := ResourcesListResult{
		Resources: []ResourceItem{},
	}
	for _, provider := range s.resources {
		items, errObj := provider.List(ctx)
		if errObj != nil {
			return errorResponse(req.ID, errObj)
		}
		if len(items) > 0 {
			result.Resources = append(result.Resources, items...)
		}
	}

	return resultResponse(req.ID, result)
}

func (s *Server) handleResourcesRead(
	ctx context.Context,
	req Request,
) *Response {
	if errResp := s.requireReady(req.ID); errResp != nil {
		return errResp
	}

	if len(req.Params) == 0 {
		return errorResponse(
			req.ID,
			newError(ErrInvalidParams, "resources/read params required", nil),
		)
	}

	if _, errResp := requireObjectParams(req.ID, req.Params); errResp != nil {
		return errResp
	}

	var params ResourcesReadParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errorResponse(
			req.ID,
			newError(ErrInvalidParams, "invalid resources/read params", nil),
		)
	}

	if params.URI == "" {
		return errorResponse(
			req.ID,
			newError(ErrInvalidParams, "resource uri required", nil),
		)
	}

	for _, provider := range s.resources {
		content, handled, errObj := provider.Read(ctx, params.URI)
		if !handled {
			continue
		}
		if errObj != nil {
			return errorResponse(req.ID, errObj)
		}
		if content == nil {
			return errorResponse(
				req.ID,
				newError(ErrInvalidParams, "resource not found", nil),
			)
		}
		return resultResponse(req.ID, ResourcesReadResult{
			Contents: []ResourceContent{*content},
		})
	}

	return errorResponse(
		req.ID,
		newError(ErrInvalidParams, "resource not found", nil),
	)
}

func (s *Server) handlePing(
	ctx context.Context,
	req Request,
) *Response {
	if errResp := s.requireReady(req.ID); errResp != nil {
		return errResp
	}

	if len(req.Params) > 0 {
		params, errResp := requireObjectParams(req.ID, req.Params)
		if errResp != nil {
			return errResp
		}
		if len(params) > 0 {
			return errorResponse(
				req.ID,
				newError(ErrInvalidParams, "ping params not supported", nil),
			)
		}
	}

	_ = ctx

	return resultResponse(req.ID, map[string]interface{}{})
}

func (s *Server) requireReady(id *json.RawMessage) *Response {
	if s.getState() == stateReady {
		return nil
	}
	return errorResponse(
		id,
		newError(ErrInvalidRequest, "server not initialized", nil),
	)
}

func (s *Server) handleNotification(ctx context.Context, req Request) {
	switch req.Method {
	case "initialized":
		s.handleInitializedNotification()
	case "notifications/initialized":
		s.handleInitializedNotification()
	case "notifications/cancelled":
		s.logger.Warn("request cancelled notification received")
	case "ping":
		s.logger.Debug("ping notification ignored")
	default:
		s.logger.Warn("notification ignored", "method", req.Method)
	}
	_ = ctx
}

func (s *Server) handleInitializedNotification() {
	switch s.getState() {
	case stateInitialized:
		s.transitionFrom(stateInitialized, stateReady)
	case stateReady:
		// Repeated initialized notifications are safe to ignore.
	default:
		s.logger.Warn("initialized notification received before initialize")
	}
}

func (s *Server) listTools() []ToolDefinition {
	definitions := make([]ToolDefinition, 0, len(s.tools))
	for _, registration := range s.tools {
		definitions = append(definitions, registration.Definition)
	}
	return definitions
}

func (s *Server) transitionFrom(from, to connectionState) bool {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if s.state != from {
		return false
	}
	s.state = to
	return true
}

func (s *Server) getState() connectionState {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	return s.state
}

func isValidRequest(req Request) bool {
	if req.JSONRPC != JSONRPCVersion {
		return false
	}
	if req.Method == "" {
		return false
	}
	if req.IDSet && req.ID == nil {
		return false
	}
	if req.ID != nil {
		if len(*req.ID) == 0 {
			return false
		}
		if !isValidID(*req.ID) {
			return false
		}
	}
	return true
}

func isValidID(raw json.RawMessage) bool {
	var value interface{}
	if err := json.Unmarshal(raw, &value); err != nil {
		return false
	}

	switch value.(type) {
	case string:
		return true
	case float64:
		return true
	default:
		return false
	}
}

func requireObjectParams(
	id *json.RawMessage,
	raw json.RawMessage,
) (map[string]json.RawMessage, *Response) {
	var params map[string]json.RawMessage
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, errorResponse(
			id,
			newError(ErrInvalidParams, "params must be an object", nil),
		)
	}
	if params == nil {
		return nil, errorResponse(
			id,
			newError(ErrInvalidParams, "params must be an object", nil),
		)
	}
	return params, nil
}

func parseErrorResponse() Response {
	return Response{
		JSONRPC: JSONRPCVersion,
		ID:      json.RawMessage("null"),
		Error:   newError(ErrParse, "parse error", nil),
	}
}

func invalidRequestResponse(id *json.RawMessage) *Response {
	return errorResponse(
		id,
		newError(ErrInvalidRequest, "invalid request", nil),
	)
}

func methodNotFoundResponse(id *json.RawMessage, method string) *Response {
	return errorResponse(
		id,
		newError(ErrMethodNotFound, "method not found", map[string]string{
			"method": method,
		}),
	)
}

func errorResponse(id *json.RawMessage, errObj *ErrorObject) *Response {
	responseID := json.RawMessage("null")
	if id != nil {
		responseID = *id
	}
	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      responseID,
		Error:   errObj,
	}
}

func resultResponse(id *json.RawMessage, result interface{}) *Response {
	responseID := json.RawMessage("null")
	if id != nil {
		responseID = *id
	}
	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      responseID,
		Result:  result,
	}
}

func (s *Server) encodeResponses(
	responses []Response,
) ([][]byte, error) {
	if len(responses) == 0 {
		return nil, nil
	}

	var payload []byte
	var err error
	if len(responses) == 1 {
		payload, err = json.Marshal(responses[0])
	} else {
		payload, err = json.Marshal(responses)
	}

	if err != nil {
		return nil, err
	}

	return [][]byte{payload}, nil
}
