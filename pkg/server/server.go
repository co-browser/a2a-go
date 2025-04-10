package server

import (
	"a2a-go/pkg/types"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// A2AServer represents an A2A server that handles JSON-RPC requests
type A2AServer struct {
	host         string
	port         int
	endpoint     string
	taskManager  TaskManager
	agentCard    *types.AgentCard
	server       *http.Server
}

// NewA2AServer creates a new A2AServer instance
func NewA2AServer(host string, port int, endpoint string, agentCard *types.AgentCard, taskManager TaskManager) (*A2AServer, error) {
	if agentCard == nil {
		return nil, fmt.Errorf("agent_card is not defined")
	}

	if taskManager == nil {
		return nil, fmt.Errorf("task_manager is not defined")
	}

	return &A2AServer{
		host:        host,
		port:        port,
		endpoint:    endpoint,
		agentCard:   agentCard,
		taskManager: taskManager,
	}, nil
}

// Start starts the A2A server
func (s *A2AServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc(s.endpoint, s.processRequest)
	mux.HandleFunc("/.well-known/agent.json", s.getAgentCard)

	s.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.host, s.port),
		Handler: mux,
	}

	log.Printf("Starting server on %s:%d", s.host, s.port)
	return s.server.ListenAndServe()
}

// getAgentCard handles requests for the agent card
func (s *A2AServer) getAgentCard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(s.agentCard); err != nil {
		http.Error(w, "Failed to encode agent card", http.StatusInternalServerError)
		return
	}
}

// processRequest handles JSON-RPC requests
func (s *A2AServer) processRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var jsonRPCRequest types.JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&jsonRPCRequest); err != nil {
		s.handleError(w, &types.JSONRPCError{
			Code:    -32700,
			Message: "Parse error",
		})
		return
	}

	var result interface{}
	var err error

	switch jsonRPCRequest.Method {
	case "get_task":
		result = s.taskManager.OnGetTask(&jsonRPCRequest)
	case "send_task":
		result = s.taskManager.OnSendTask(&jsonRPCRequest)
	case "send_task_streaming":
		result, err = s.taskManager.OnSendTaskSubscribe(&jsonRPCRequest)
	case "cancel_task":
		result = s.taskManager.OnCancelTask(&jsonRPCRequest)
	case "set_task_push_notification":
		result = s.taskManager.OnSetTaskPushNotification(&jsonRPCRequest)
	case "get_task_push_notification":
		result = s.taskManager.OnGetTaskPushNotification(&jsonRPCRequest)
	case "resubscribe_to_task":
		result, err = s.taskManager.OnResubscribeToTask(&jsonRPCRequest)
	default:
		s.handleError(w, &types.JSONRPCError{
			Code:    -32601,
			Message: "Method not found",
		})
		return
	}

	if err != nil {
		s.handleError(w, &types.JSONRPCError{
			Code:    -32603,
			Message: err.Error(),
		})
		return
	}

	s.createResponse(w, result)
}

// handleError handles error responses
func (s *A2AServer) handleError(w http.ResponseWriter, error *types.JSONRPCError) {
	response := &types.JSONRPCResponse{
		JSONRPC: "2.0",
		Error:   error,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode error response: %v", err)
	}
}

// createResponse creates the appropriate response based on the result type
func (s *A2AServer) createResponse(w http.ResponseWriter, result interface{}) {
	w.Header().Set("Content-Type", "application/json")

	switch v := result.(type) {
	case *types.JSONRPCResponse:
		if err := json.NewEncoder(w).Encode(v); err != nil {
			log.Printf("Failed to encode JSON-RPC response: %v", err)
		}
	case chan *types.SendTaskStreamingResponse:
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		for response := range v {
			data, err := json.Marshal(response)
			if err != nil {
				log.Printf("Failed to marshal streaming response: %v", err)
				continue
			}

			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	default:
		log.Printf("Unexpected result type: %T", result)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}