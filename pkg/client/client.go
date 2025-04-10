package client

import (
	"a2a-go/pkg/types"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// A2AClient represents an A2A client for interacting with A2A servers
type A2AClient struct {
	url string
}

// NewA2AClient creates a new A2AClient instance
func NewA2AClient(agentCard *types.AgentCard, url string) (*A2AClient, error) {
	if agentCard != nil {
		return &A2AClient{url: agentCard.URL}, nil
	}
	if url != "" {
		return &A2AClient{url: url}, nil
	}
	return nil, fmt.Errorf("must provide either agent_card or url")
}

// SendTask sends a task to the A2A server
func (c *A2AClient) SendTask(payload map[string]interface{}) (*types.SendTaskResponse, error) {
	request := &types.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "send_task",
		Params:  payload,
	}

	response, err := c.sendRequest(request)
	if err != nil {
		return nil, err
	}

	var result types.SendTaskResponse
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, &types.A2AClientJSONError{
			Message: fmt.Sprintf("failed to parse response: %v", err),
		}
	}

	return &result, nil
}

// SendTaskStreaming sends a task and streams the response
func (c *A2AClient) SendTaskStreaming(payload map[string]interface{}) (chan *types.SendTaskStreamingResponse, error) {
	request := &types.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "send_task_streaming",
		Params:  payload,
	}

	client := &http.Client{
		Timeout: 0, // No timeout for streaming
	}

	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, &types.A2AClientJSONError{
			Message: fmt.Sprintf("failed to marshal request: %v", err),
		}
	}

	req, err := http.NewRequest("POST", c.url, nil)
	if err != nil {
		return nil, &types.A2AClientHTTPError{
			StatusCode: 400,
			Message:    fmt.Sprintf("failed to create request: %v", err),
		}
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Content-Type", "application/json")
	req.Body = io.NopCloser(bytes.NewReader(reqBody))

	resp, err := client.Do(req)
	if err != nil {
		return nil, &types.A2AClientHTTPError{
			StatusCode: 400,
			Message:    fmt.Sprintf("failed to send request: %v", err),
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &types.A2AClientHTTPError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("unexpected status code: %d", resp.StatusCode),
		}
	}

	responseChan := make(chan *types.SendTaskStreamingResponse)

	go func() {
		defer close(responseChan)
		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)
		for {
			var response types.SendTaskStreamingResponse
			if err := decoder.Decode(&response); err != nil {
				if err == io.EOF {
					break
				}
				responseChan <- &types.SendTaskStreamingResponse{
					Error: &types.JSONRPCError{
						Code:    500,
						Message: fmt.Sprintf("failed to decode response: %v", err),
					},
				}
				break
			}
			responseChan <- &response
		}
	}()

	return responseChan, nil
}

// sendRequest sends a JSON-RPC request to the A2A server
func (c *A2AClient) sendRequest(request *types.JSONRPCRequest) ([]byte, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, &types.A2AClientJSONError{
			Message: fmt.Sprintf("failed to marshal request: %v", err),
		}
	}

	resp, err := client.Post(c.url, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, &types.A2AClientHTTPError{
			StatusCode: 400,
			Message:    fmt.Sprintf("failed to send request: %v", err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &types.A2AClientHTTPError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("unexpected status code: %d", resp.StatusCode),
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &types.A2AClientHTTPError{
			StatusCode: 500,
			Message:    fmt.Sprintf("failed to read response: %v", err),
		}
	}

	return body, nil
}

// GetTask retrieves a task from the A2A server
func (c *A2AClient) GetTask(payload map[string]interface{}) (*types.GetTaskResponse, error) {
	request := &types.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "get_task",
		Params:  payload,
	}

	response, err := c.sendRequest(request)
	if err != nil {
		return nil, err
	}

	var result types.GetTaskResponse
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, &types.A2AClientJSONError{
			Message: fmt.Sprintf("failed to parse response: %v", err),
		}
	}

	return &result, nil
}

// CancelTask cancels a task on the A2A server
func (c *A2AClient) CancelTask(payload map[string]interface{}) (*types.CancelTaskResponse, error) {
	request := &types.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "cancel_task",
		Params:  payload,
	}

	response, err := c.sendRequest(request)
	if err != nil {
		return nil, err
	}

	var result types.CancelTaskResponse
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, &types.A2AClientJSONError{
			Message: fmt.Sprintf("failed to parse response: %v", err),
		}
	}

	return &result, nil
}

// SetTaskCallback sets a callback for a task
func (c *A2AClient) SetTaskCallback(payload map[string]interface{}) (*types.SetTaskPushNotificationResponse, error) {
	request := &types.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "set_task_push_notification",
		Params:  payload,
	}

	response, err := c.sendRequest(request)
	if err != nil {
		return nil, err
	}

	var result types.SetTaskPushNotificationResponse
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, &types.A2AClientJSONError{
			Message: fmt.Sprintf("failed to parse response: %v", err),
		}
	}

	return &result, nil
}

// GetTaskCallback retrieves a task's callback configuration
func (c *A2AClient) GetTaskCallback(payload map[string]interface{}) (*types.GetTaskPushNotificationResponse, error) {
	request := &types.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "get_task_push_notification",
		Params:  payload,
	}

	response, err := c.sendRequest(request)
	if err != nil {
		return nil, err
	}

	var result types.GetTaskPushNotificationResponse
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, &types.A2AClientJSONError{
			Message: fmt.Sprintf("failed to parse response: %v", err),
		}
	}

	return &result, nil
} 