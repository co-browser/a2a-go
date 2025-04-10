package types

import (
	"fmt"
)

// Enums
type TaskState string

const (
	TaskSubmitted   TaskState = "submitted"
	TaskWorking     TaskState = "working"
	TaskInputNeeded TaskState = "input-required"
	TaskCompleted   TaskState = "completed"
	TaskCanceled    TaskState = "canceled"
	TaskFailed      TaskState = "failed"
	TaskUnknown     TaskState = "unknown"
)

// Basic structs
type TextPart struct {
	Type     string                 `json:"type"`
	Text     string                 `json:"text"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type FileContent struct {
	Name     *string `json:"name,omitempty"`
	MimeType *string `json:"mimeType,omitempty"`
	Bytes    *string `json:"bytes,omitempty"`
	URI      *string `json:"uri,omitempty"`
}

type FilePart struct {
	Type     string       `json:"type"`
	File     FileContent  `json:"file"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type DataPart struct {
	Type     string                 `json:"type"`
	Data     map[string]interface{} `json:"data"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type Message struct {
	Role     string      `json:"role"`
	Parts    []any       `json:"parts"` // Simplified union of TextPart/FilePart/DataPart
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type TaskStatus struct {
	State     TaskState `json:"state"`
	Message   *Message  `json:"message,omitempty"`
	Timestamp string    `json:"timestamp"`
}

type Artifact struct {
	Name        *string     `json:"name,omitempty"`
	Description *string     `json:"description,omitempty"`
	Parts       []any       `json:"parts"` // Simplified
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Index       int         `json:"index"`
	Append      *bool       `json:"append,omitempty"`
	LastChunk   *bool       `json:"lastChunk,omitempty"`
}

type Task struct {
	ID        string                 `json:"id"`
	SessionID *string                `json:"sessionId,omitempty"`
	Status    TaskStatus             `json:"status"`
	Artifacts []Artifact             `json:"artifacts,omitempty"`
	History   []Message              `json:"history,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type TaskStatusUpdateEvent struct {
	ID       string                 `json:"id"`
	Status   TaskStatus             `json:"status"`
	Final    bool                   `json:"final"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type TaskArtifactUpdateEvent struct {
	ID       string                 `json:"id"`
	Artifact Artifact               `json:"artifact"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Push Notifications
type AuthenticationInfo struct {
	Schemes     []string `json:"schemes"`
	Credentials *string  `json:"credentials,omitempty"`
}

type PushNotificationConfig struct {
	URL           string             `json:"url"`
	Token         *string            `json:"token,omitempty"`
	Authentication *AuthenticationInfo `json:"authentication,omitempty"`
}

type TaskIdParams struct {
	ID       string                 `json:"id"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type TaskQueryParams struct {
	TaskIdParams
	HistoryLength *int `json:"historyLength,omitempty"`
}

type TaskSendParams struct {
	ID                string                  `json:"id"`
	SessionID         string                  `json:"sessionId"`
	Message           Message                 `json:"message"`
	AcceptedOutputModes []string              `json:"acceptedOutputModes,omitempty"`
	PushNotification  *PushNotificationConfig `json:"pushNotification,omitempty"`
	HistoryLength     *int                    `json:"historyLength,omitempty"`
	Metadata          map[string]interface{}  `json:"metadata,omitempty"`
}

type TaskPushNotificationConfig struct {
	ID                    string                 `json:"id"`
	PushNotificationConfig PushNotificationConfig `json:"pushNotificationConfig"`
}

// JSON-RPC
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

type SendTaskResponse struct {
	Result *Task `json:"result,omitempty"`
}

type SendTaskStreamingResponse struct {
	Result interface{} `json:"result,omitempty"`
	Error  *JSONRPCError `json:"error,omitempty"`
	ID     interface{} `json:"id"`
}

type CancelTaskResponse struct {
	Result *Task `json:"result,omitempty"`
}

type GetTaskResponse struct {
	Result *Task `json:"result,omitempty"`
}

type SetTaskPushNotificationResponse struct {
	Result *TaskPushNotificationConfig `json:"result,omitempty"`
}

type GetTaskPushNotificationResponse struct {
	Result *TaskPushNotificationConfig `json:"result,omitempty"`
}

// Agent info
type AgentProvider struct {
	Organization string  `json:"organization"`
	URL          *string `json:"url,omitempty"`
}

type AgentCapabilities struct {
	Streaming             bool `json:"streaming"`
	PushNotifications     bool `json:"pushNotifications"`
	StateTransitionHistory bool `json:"stateTransitionHistory"`
}

type AgentAuthentication struct {
	Schemes     []string `json:"schemes"`
	Credentials *string  `json:"credentials,omitempty"`
}

type AgentSkill struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Description   *string  `json:"description,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Examples      []string `json:"examples,omitempty"`
	InputModes    []string `json:"inputModes,omitempty"`
	OutputModes   []string `json:"outputModes,omitempty"`
}

type AgentCard struct {
	Name               string               `json:"name"`
	Description        *string              `json:"description,omitempty"`
	URL                string               `json:"url"`
	Provider           *AgentProvider       `json:"provider,omitempty"`
	Version            string               `json:"version"`
	DocumentationURL   *string              `json:"documentationUrl,omitempty"`
	Capabilities       AgentCapabilities    `json:"capabilities"`
	Authentication     *AgentAuthentication `json:"authentication,omitempty"`
	DefaultInputModes  []string             `json:"defaultInputModes"`
	DefaultOutputModes []string             `json:"defaultOutputModes"`
	Skills             []AgentSkill         `json:"skills"`
}

// A2AClientJSONError represents a JSON parsing error in the A2A client
type A2AClientJSONError struct {
	Message string
}

func (e *A2AClientJSONError) Error() string {
	return e.Message
}

// A2AClientHTTPError represents an HTTP error in the A2A client
type A2AClientHTTPError struct {
	StatusCode int
	Message    string
}

func (e *A2AClientHTTPError) Error() string {
	return fmt.Sprintf("HTTP error %d: %s", e.StatusCode, e.Message)
}
