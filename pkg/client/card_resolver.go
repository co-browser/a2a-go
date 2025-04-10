package client

import (
	"a2a-go/pkg/types"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// A2ACardResolver handles fetching and parsing agent cards from A2A servers
type A2ACardResolver struct {
	baseURL       string
	agentCardPath string
	client        *http.Client
}

// NewA2ACardResolver creates a new A2ACardResolver instance
func NewA2ACardResolver(baseURL, agentCardPath string) *A2ACardResolver {
	// Clean up the URLs
	baseURL = strings.TrimRight(baseURL, "/")
	agentCardPath = strings.TrimLeft(agentCardPath, "/")

	return &A2ACardResolver{
		baseURL:       baseURL,
		agentCardPath: agentCardPath,
		client:        &http.Client{},
	}
}

// GetAgentCard fetches and parses the agent card from the A2A server
func (r *A2ACardResolver) GetAgentCard() (*types.AgentCard, error) {
	url := fmt.Sprintf("%s/%s", r.baseURL, r.agentCardPath)
	
	resp, err := r.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch agent card: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch agent card: status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var card types.AgentCard
	if err := json.Unmarshal(body, &card); err != nil {
		return nil, &types.A2AClientJSONError{
			Message: fmt.Sprintf("failed to parse agent card JSON: %v", err),
		}
	}

	return &card, nil
} 