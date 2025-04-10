package server

import (
	"errors"
	"sync"
	"time"

	"a2a-go/pkg/types"
)

// TaskManager defines the interface for task management operations
type TaskManager interface {
	OnGetTask(request *types.JSONRPCRequest) *types.GetTaskResponse
	OnCancelTask(request *types.JSONRPCRequest) *types.CancelTaskResponse
	OnSendTask(request *types.JSONRPCRequest) *types.SendTaskResponse
	OnSendTaskSubscribe(request *types.JSONRPCRequest) (*types.SendTaskStreamingResponse, error)
	OnSetTaskPushNotification(request *types.JSONRPCRequest) *types.SetTaskPushNotificationResponse
	OnGetTaskPushNotification(request *types.JSONRPCRequest) *types.GetTaskPushNotificationResponse
	OnResubscribeToTask(request *types.JSONRPCRequest) (*types.SendTaskStreamingResponse, error)
}

// InMemoryTaskManager implements TaskManager with in-memory storage
type InMemoryTaskManager struct {
	tasks                  map[string]*types.Task
	pushNotificationInfos  map[string]*types.PushNotificationConfig
	lock                  sync.Mutex
	taskSSESubscribers    map[string][]chan interface{}
	subscriberLock        sync.Mutex
}

// NewInMemoryTaskManager creates a new instance of InMemoryTaskManager
func NewInMemoryTaskManager() *InMemoryTaskManager {
	return &InMemoryTaskManager{
		tasks:                 make(map[string]*types.Task),
		pushNotificationInfos: make(map[string]*types.PushNotificationConfig),
		taskSSESubscribers:    make(map[string][]chan interface{}),
	}
}

// OnGetTask handles task retrieval requests
func (tm *InMemoryTaskManager) OnGetTask(request *types.JSONRPCRequest) *types.GetTaskResponse {
	taskQueryParams := request.Params.(*types.TaskQueryParams)

	tm.lock.Lock()
	task := tm.tasks[taskQueryParams.ID]
	tm.lock.Unlock()

	if task == nil {
		return &types.GetTaskResponse{
			Result: nil,
		}
	}

	taskResult := tm.appendTaskHistory(task, taskQueryParams.HistoryLength)
	return &types.GetTaskResponse{
		Result: taskResult,
	}
}

// OnCancelTask handles task cancellation requests
func (tm *InMemoryTaskManager) OnCancelTask(request *types.JSONRPCRequest) *types.CancelTaskResponse {
	taskIDParams := request.Params.(*types.TaskIdParams)

	tm.lock.Lock()
	task := tm.tasks[taskIDParams.ID]
	tm.lock.Unlock()

	if task == nil {
		return &types.CancelTaskResponse{
			Result: nil,
		}
	}

	return &types.CancelTaskResponse{
		Result: nil,
	}
}

// OnSendTask handles task submission requests
func (tm *InMemoryTaskManager) OnSendTask(request *types.JSONRPCRequest) *types.SendTaskResponse {
	// To be implemented by concrete implementation
	return nil
}

// OnSendTaskSubscribe handles task subscription requests
func (tm *InMemoryTaskManager) OnSendTaskSubscribe(request *types.JSONRPCRequest) (*types.SendTaskStreamingResponse, error) {
	// To be implemented by concrete implementation
	return nil, errors.New("not implemented")
}

// setPushNotificationInfo sets push notification configuration for a task
func (tm *InMemoryTaskManager) setPushNotificationInfo(taskID string, notificationConfig *types.PushNotificationConfig) error {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	task := tm.tasks[taskID]
	if task == nil {
		return errors.New("task not found")
	}

	tm.pushNotificationInfos[taskID] = notificationConfig
	return nil
}

// getPushNotificationInfo retrieves push notification configuration for a task
func (tm *InMemoryTaskManager) getPushNotificationInfo(taskID string) (*types.PushNotificationConfig, error) {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	task := tm.tasks[taskID]
	if task == nil {
		return nil, errors.New("task not found")
	}

	return tm.pushNotificationInfos[taskID], nil
}

// hasPushNotificationInfo checks if a task has push notification configuration
func (tm *InMemoryTaskManager) hasPushNotificationInfo(taskID string) bool {
	tm.lock.Lock()
	defer tm.lock.Unlock()
	_, exists := tm.pushNotificationInfos[taskID]
	return exists
}

// OnSetTaskPushNotification handles setting push notification configuration
func (tm *InMemoryTaskManager) OnSetTaskPushNotification(request *types.JSONRPCRequest) *types.SetTaskPushNotificationResponse {
	taskNotificationParams := request.Params.(*types.TaskPushNotificationConfig)

	err := tm.setPushNotificationInfo(taskNotificationParams.ID, &taskNotificationParams.PushNotificationConfig)
	if err != nil {
		return &types.SetTaskPushNotificationResponse{
			Result: nil,
		}
	}

	return &types.SetTaskPushNotificationResponse{
		Result: taskNotificationParams,
	}
}

// OnGetTaskPushNotification handles retrieving push notification configuration
func (tm *InMemoryTaskManager) OnGetTaskPushNotification(request *types.JSONRPCRequest) *types.GetTaskPushNotificationResponse {
	taskParams := request.Params.(*types.TaskIdParams)

	notificationInfo, err := tm.getPushNotificationInfo(taskParams.ID)
	if err != nil {
		return &types.GetTaskPushNotificationResponse{
			Result: nil,
		}
	}

	return &types.GetTaskPushNotificationResponse{
		Result: &types.TaskPushNotificationConfig{
			ID:                    taskParams.ID,
			PushNotificationConfig: *notificationInfo,
		},
	}
}

// upsertTask creates or updates a task
func (tm *InMemoryTaskManager) upsertTask(taskSendParams *types.TaskSendParams) *types.Task {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	task := tm.tasks[taskSendParams.ID]
	if task == nil {
		task = &types.Task{
			ID:       taskSendParams.ID,
			SessionID: &taskSendParams.SessionID,
			Status: types.TaskStatus{
				State:     types.TaskSubmitted,
				Timestamp: time.Now().Format(time.RFC3339),
			},
			History: []types.Message{taskSendParams.Message},
		}
		tm.tasks[taskSendParams.ID] = task
	} else {
		task.History = append(task.History, taskSendParams.Message)
	}

	return task
}

// OnResubscribeToTask handles task resubscription requests
func (tm *InMemoryTaskManager) OnResubscribeToTask(request *types.JSONRPCRequest) (*types.SendTaskStreamingResponse, error) {
	return nil, errors.New("not implemented")
}

// updateStore updates task status and artifacts
func (tm *InMemoryTaskManager) updateStore(taskID string, status types.TaskStatus, artifacts []types.Artifact) (*types.Task, error) {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	task := tm.tasks[taskID]
	if task == nil {
		return nil, errors.New("task not found")
	}

	task.Status = status

	if status.Message != nil {
		task.History = append(task.History, *status.Message)
	}

	if artifacts != nil {
		if task.Artifacts == nil {
			task.Artifacts = []types.Artifact{}
		}
		task.Artifacts = append(task.Artifacts, artifacts...)
	}

	return task, nil
}

// appendTaskHistory limits task history to specified length
func (tm *InMemoryTaskManager) appendTaskHistory(task *types.Task, historyLength *int) *types.Task {
	newTask := *task
	if historyLength != nil && *historyLength > 0 {
		if len(newTask.History) > *historyLength {
			newTask.History = newTask.History[len(newTask.History)-*historyLength:]
		}
	} else {
		newTask.History = []types.Message{}
	}
	return &newTask
}

// setupSSEConsumer sets up SSE consumer for a task
func (tm *InMemoryTaskManager) setupSSEConsumer(taskID string, isResubscribe bool) (chan interface{}, error) {
	tm.subscriberLock.Lock()
	defer tm.subscriberLock.Unlock()

	if _, exists := tm.taskSSESubscribers[taskID]; !exists {
		if isResubscribe {
			return nil, errors.New("task not found for resubscription")
		}
		tm.taskSSESubscribers[taskID] = []chan interface{}{}
	}

	sseEventQueue := make(chan interface{})
	tm.taskSSESubscribers[taskID] = append(tm.taskSSESubscribers[taskID], sseEventQueue)
	return sseEventQueue, nil
}

// enqueueEventsForSSE sends events to SSE subscribers
func (tm *InMemoryTaskManager) enqueueEventsForSSE(taskID string, taskUpdateEvent interface{}) {
	tm.subscriberLock.Lock()
	subscribers, exists := tm.taskSSESubscribers[taskID]
	tm.subscriberLock.Unlock()

	if !exists {
		return
	}

	for _, subscriber := range subscribers {
		subscriber <- taskUpdateEvent
	}
}

// dequeueEventsForSSE processes events from SSE queue
func (tm *InMemoryTaskManager) dequeueEventsForSSE(requestID interface{}, taskID string, sseEventQueue chan interface{}) chan *types.SendTaskStreamingResponse {
	responseChan := make(chan *types.SendTaskStreamingResponse)

	go func() {
		defer close(responseChan)
		defer func() {
			tm.subscriberLock.Lock()
			if subscribers, exists := tm.taskSSESubscribers[taskID]; exists {
				for i, sub := range subscribers {
					if sub == sseEventQueue {
						tm.taskSSESubscribers[taskID] = append(subscribers[:i], subscribers[i+1:]...)
						break
					}
				}
			}
			tm.subscriberLock.Unlock()
		}()

		for {
			event, ok := <-sseEventQueue
			if !ok {
				break
			}

			if err, isError := event.(*types.JSONRPCError); isError {
				responseChan <- &types.SendTaskStreamingResponse{
					ID:    requestID,
					Error: err,
				}
				break
			}

			responseChan <- &types.SendTaskStreamingResponse{
				ID:     requestID,
				Result: event,
			}

			if statusEvent, isStatusEvent := event.(*types.TaskStatusUpdateEvent); isStatusEvent && statusEvent.Final {
				break
			}
		}
	}()

	return responseChan
}
