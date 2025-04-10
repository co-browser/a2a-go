# A2A Go

<strong>A Go implementation of the [Agent2Agent (A2A) protocol](https://github.com/google/A2A#agent2agent-protocol-a2a), enabling seamless communication between different agent frameworks.</strong>


> [!CAUTION]
> This is an early development effort, do not use in production


## Installation

```bash
go get github.com/co-browser/a2a-go
```

## Quickstart

First, we'll start one of the google A2A agents:

```
cd A2A/samples/python/agents
uv run google_adk/.
INFO:     Started server process [4070]
INFO:     Waiting for application startup.
INFO:     Application startup complete.
INFO:     Uvicorn running on http://localhost:10002 (Press CTRL+C to quit)
```

then we'll connect to it with a2a-go:

```
go run cmd/cli/main.go -agent=http://localhost:10002
======= Agent Card ========
{
  "name": "Reimbursement Agent",
  "description": "This agent handles the reimbursement process for the employees given the amount and purpose of the reimbursement.",
  "url": "http://localhost:10002/",
  "version": "1.0.0",
  "capabilities": {
    "streaming": true,
    "pushNotifications": false,
    "stateTransitionHistory": false
  },
  "defaultInputModes": [
    "text",
    "text/plain"
  ],
  "defaultOutputModes": [
    "text",
    "text/plain"
  ],
  "skills": [
    {
      "id": "process_reimbursement",
      "name": "Process Reimbursement Tool",
      "description": "Helps with the reimbursement process for users given the amount and purpose of the reimbursement.",
      "tags": [
        "reimbursement"
      ],
      "examples": [
        "Can you reimburse me $20 for my lunch with the clients?"
      ]
    }
  ]
}
=========  starting a new task ========

What do you want to send to the agent? (:q or quit to exit)
```

## Documentation

coming soon
