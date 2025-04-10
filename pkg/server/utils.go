package server

import (
	"a2a-go/pkg/types"
)

// AreModalitiesCompatible checks if server and client output modes are compatible
// Modalities are compatible if they are both non-empty and there is at least one common element
func AreModalitiesCompatible(serverOutputModes, clientOutputModes []string) bool {
	if len(clientOutputModes) == 0 {
		return true
	}

	if len(serverOutputModes) == 0 {
		return true
	}

	// Create a map for faster lookup
	serverModes := make(map[string]struct{})
	for _, mode := range serverOutputModes {
		serverModes[mode] = struct{}{}
	}

	// Check if any client mode exists in server modes
	for _, mode := range clientOutputModes {
		if _, exists := serverModes[mode]; exists {
			return true
		}
	}

	return false
}

// NewIncompatibleTypesError creates a new JSONRPCResponse with ContentTypeNotSupportedError
func NewIncompatibleTypesError(requestID interface{}) *types.JSONRPCResponse {
	return &types.JSONRPCResponse{
		ID: requestID,
		Error: &types.JSONRPCError{
			Code:    415,
			Message: "Content type not supported",
		},
	}
}

// NewNotImplementedError creates a new JSONRPCResponse with UnsupportedOperationError
func NewNotImplementedError(requestID interface{}) *types.JSONRPCResponse {
	return &types.JSONRPCResponse{
		ID: requestID,
		Error: &types.JSONRPCError{
			Code:    501,
			Message: "Operation not implemented",
		},
	}
} 