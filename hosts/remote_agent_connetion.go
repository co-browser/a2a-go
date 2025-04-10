package hosts

import (
	"a2a-go/pkg/client"
	"a2a-go/pkg/types"
	"context"
	"fmt"
	"github.com/google/uuid"
	"sync"
)

// TaskCallbackArg represents the possible types that can be passed to a task callback
type TaskCallbackArg interface {
	IsTaskCallbackArg()
	Unwrap() interface{}
}

// TaskWrapper wraps types.Task to implement TaskCallbackArg
type TaskWrapper struct {
	*types.Task
}

func (t TaskWrapper) IsTaskCallbackArg() {}
func (t TaskWrapper) Unwrap() interface{} { return t.Task }

// TaskStatusUpdateWrapper wraps types.TaskStatusUpdateEvent to implement TaskCallbackArg
type TaskStatusUpdateWrapper struct {
	*types.TaskStatusUpdateEvent
}

func (t TaskStatusUpdateWrapper) IsTaskCallbackArg() {}
func (t TaskStatusUpdateWrapper) Unwrap() interface{} { return t.TaskStatusUpdateEvent }

// TaskArtifactUpdateWrapper wraps types.TaskArtifactUpdateEvent to implement TaskCallbackArg
type TaskArtifactUpdateWrapper struct {
	*types.TaskArtifactUpdateEvent
}

func (t TaskArtifactUpdateWrapper) IsTaskCallbackArg() {}
func (t TaskArtifactUpdateWrapper) Unwrap() interface{} { return t.TaskArtifactUpdateEvent }

// TaskUpdateCallback is a function type that handles task updates
type TaskUpdateCallback func(TaskCallbackArg) *types.Task

// RemoteAgentConnections holds the connections to remote agents
type RemoteAgentConnections struct {
	agentClient *client.A2AClient
	card        *types.AgentCard

	conversationName string
	pendingTasks     sync.Map // Using sync.Map for thread-safe set operations
}

// NewRemoteAgentConnections creates a new RemoteAgentConnections instance
func NewRemoteAgentConnections(agentCard *types.AgentCard) (*RemoteAgentConnections, error) {
	client, err := client.NewA2AClient(agentCard, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create A2A client: %v", err)
	}
	return &RemoteAgentConnections{
		agentClient: client,
		card:        agentCard,
	}, nil
}

// GetAgent returns the agent card
func (r *RemoteAgentConnections) GetAgent() *types.AgentCard {
	return r.card
}

// SendTask sends a task to the remote agent
func (r *RemoteAgentConnections) SendTask(ctx context.Context, request *types.TaskSendParams, taskCallback TaskUpdateCallback) (*types.Task, error) {
	if r.card.Capabilities.Streaming {
		var task *types.Task
		if taskCallback != nil {
			initialTask := &types.Task{
				ID:        request.ID,
				SessionID: &request.SessionID,
				Status: types.TaskStatus{
					State:   types.TaskSubmitted,
					Message: &request.Message,
				},
				History: []types.Message{request.Message},
			}
			task = taskCallback(TaskWrapper{initialTask})
		}

		// Create a channel to receive streaming responses
		responseChan := make(chan *types.Task)
		errorChan := make(chan error)

		// Start streaming in a goroutine
		go func() {
			streamChan, err := r.agentClient.SendTaskStreaming(request.Metadata)
			if err != nil {
				errorChan <- fmt.Errorf("failed to start streaming: %v", err)
				return
			}

			for response := range streamChan {
				if response.Error != nil {
					errorChan <- fmt.Errorf("streaming error: %v", response.Error)
					return
				}

				if taskResult, ok := response.Result.(*types.Task); ok {
					mergeMetadata(taskResult, request)
					
					// Handle task status updates
					if taskResult.Status.Message != nil {
						mergeMetadata(taskResult.Status.Message, request.Message)
						if taskResult.Status.Message.Metadata == nil {
							taskResult.Status.Message.Metadata = make(map[string]interface{})
						}
						if messageID, exists := taskResult.Status.Message.Metadata["message_id"]; exists {
							taskResult.Status.Message.Metadata["last_message_id"] = messageID
						}
						taskResult.Status.Message.Metadata["message_id"] = uuid.New().String()
					}

					if taskCallback != nil {
						task = taskCallback(TaskWrapper{taskResult})
					}
				} else if statusUpdate, ok := response.Result.(*types.TaskStatusUpdateEvent); ok {
					if statusUpdate.Final {
						responseChan <- task
						return
					}
				}
			}
		}()

		// Wait for either the final response or an error
		select {
		case result := <-responseChan:
			return result, nil
		case err := <-errorChan:
			return nil, err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	} else {
		// Non-streaming case
		response, err := r.agentClient.SendTask(request.Metadata)
		if err != nil {
			return nil, err
		}

		if response.Result != nil {
			mergeMetadata(response.Result, request)

			// Handle task status updates
			if response.Result.Status.Message != nil {
				mergeMetadata(response.Result.Status.Message, request.Message)
				if response.Result.Status.Message.Metadata == nil {
					response.Result.Status.Message.Metadata = make(map[string]interface{})
				}
				if messageID, exists := response.Result.Status.Message.Metadata["message_id"]; exists {
					response.Result.Status.Message.Metadata["last_message_id"] = messageID
				}
				response.Result.Status.Message.Metadata["message_id"] = uuid.New().String()
			}

			if taskCallback != nil {
				taskCallback(TaskWrapper{response.Result})
			}

			return response.Result, nil
		}
		return nil, fmt.Errorf("no result in response")
	}
}

// mergeMetadata merges metadata from source to target
func mergeMetadata(target, source interface{}) {
	targetMsg, ok1 := target.(*types.Message)
	sourceMsg, ok2 := source.(*types.Message)
	if ok1 && ok2 {
		if targetMsg.Metadata == nil {
			targetMsg.Metadata = make(map[string]interface{})
		}
		if sourceMsg.Metadata != nil {
			for k, v := range sourceMsg.Metadata {
				targetMsg.Metadata[k] = v
			}
		}
		return
	}

	targetTask, ok1 := target.(*types.Task)
	sourceTask, ok2 := source.(*types.TaskSendParams)
	if ok1 && ok2 {
		if targetTask.Metadata == nil {
			targetTask.Metadata = make(map[string]interface{})
		}
		if sourceTask.Metadata != nil {
			for k, v := range sourceTask.Metadata {
				targetTask.Metadata[k] = v
			}
		}
	}
} 