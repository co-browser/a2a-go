package main

import (
	"a2a-go/pkg/client"
	"a2a-go/pkg/cli"
	"a2a-go/pkg/types"
	"a2a-go/pkg/utils"
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/google/uuid"
)

type Config struct {
	agent                   string
	session                 string
	history                 bool
	usePushNotifications    bool
	pushNotificationReceiver string
}

func completeTask(
	client *client.A2AClient,
	streaming bool,
	usePushNotifications bool,
	notificationReceiverHost string,
	notificationReceiverPort string,
	taskID string,
	sessionID string,
) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nWhat do you want to send to the agent? (:q or quit to exit)\n")
	prompt, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("error reading input: %v", err)
	}

	prompt = strings.TrimSpace(prompt)
	if prompt == ":q" || prompt == "quit" {
		return false, nil
	}

	payload := map[string]interface{}{
		"id":                taskID,
		"sessionId":        sessionID,
		"acceptedOutputModes": []string{"text"},
		"message": map[string]interface{}{
			"role": "user",
			"parts": []map[string]interface{}{
				{
					"type": "text",
					"text": prompt,
				},
			},
		},
	}

	if usePushNotifications {
		payload["pushNotification"] = map[string]interface{}{
			"url": fmt.Sprintf("http://%s:%s/notify", notificationReceiverHost, notificationReceiverPort),
			"authentication": map[string]interface{}{
				"schemes": []string{"bearer"},
			},
		}
	}

	var taskResult *types.GetTaskResponse

	if streaming {
		responseChan, err := client.SendTaskStreaming(payload)
		if err != nil {
			return false, fmt.Errorf("error sending streaming task: %v", err)
		}

		for response := range responseChan {
			jsonBytes, err := json.Marshal(response)
			if err != nil {
				log.Printf("Error marshaling stream event: %v", err)
				continue
			}
			fmt.Printf("stream event => %s\n", string(jsonBytes))
		}

		taskResult, err = client.GetTask(map[string]interface{}{"id": taskID})
		if err != nil {
			return false, fmt.Errorf("error getting task: %v", err)
		}
	} else {
		sendResult, err := client.SendTask(payload)
		if err != nil {
			return false, fmt.Errorf("error sending task: %v", err)
		}

		jsonBytes, err := json.Marshal(sendResult)
		if err != nil {
			return false, fmt.Errorf("error marshaling result: %v", err)
		}
		fmt.Printf("\n%s\n", string(jsonBytes))

		taskResult = &types.GetTaskResponse{Result: sendResult.Result}
	}

	if taskResult.Result.Status.State == types.TaskInputNeeded {
		return completeTask(
			client,
			streaming,
			usePushNotifications,
			notificationReceiverHost,
			notificationReceiverPort,
			taskID,
			sessionID,
		)
	}

	return true, nil
}

func main() {
	config := Config{}
	flag.StringVar(&config.agent, "agent", "http://localhost:10000", "Agent URL")
	flag.StringVar(&config.session, "session", "", "Session ID (0 for new session)")
	flag.BoolVar(&config.history, "history", false, "Show history")
	flag.BoolVar(&config.usePushNotifications, "use-push-notifications", false, "Use push notifications")
	flag.StringVar(&config.pushNotificationReceiver, "push-notification-receiver", "http://localhost:5000", "Push notification receiver URL")
	flag.Parse()

	// Create card resolver and get agent card
	cardResolver := client.NewA2ACardResolver(config.agent, "/.well-known/agent.json")
	card, err := cardResolver.GetAgentCard()
	if err != nil {
		log.Fatalf("Error getting agent card: %v", err)
	}

	jsonBytes, err := json.MarshalIndent(card, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling agent card: %v", err)
	}
	fmt.Printf("======= Agent Card ========\n%s\n", string(jsonBytes))

	// Parse notification receiver URL
	notifReceiverURL, err := url.Parse(config.pushNotificationReceiver)
	if err != nil {
		log.Fatalf("Error parsing notification receiver URL: %v", err)
	}

	var pushNotificationListener *cli.PushNotificationListener
	if config.usePushNotifications {
		notificationReceiverAuth := &utils.PushNotificationReceiverAuth{}
		jwksURL := fmt.Sprintf("%s/.well-known/jwks.json", config.agent)
		if err := notificationReceiverAuth.LoadJWKS(jwksURL); err != nil {
			log.Fatalf("Error loading JWKS: %v", err)
		}

		pushNotificationListener = cli.NewPushNotificationListener(
			notifReceiverURL.Hostname(),
			notifReceiverURL.Port(),
			notificationReceiverAuth,
		)
		pushNotificationListener.Start()
		defer pushNotificationListener.Stop()
	}

	// Create A2A client
	a2aClient, err := client.NewA2AClient(card, "")
	if err != nil {
		log.Fatalf("Error creating A2A client: %v", err)
	}

	// Set up session ID
	sessionID := config.session
	if sessionID == "" || sessionID == "0" {
		sessionID = uuid.New().String()
	}

	continueLoop := true
	streaming := card.Capabilities.Streaming

	for continueLoop {
		taskID := uuid.New().String()
		fmt.Println("=========  starting a new task ======== ")
		
		continueLoop, err = completeTask(
			a2aClient,
			streaming,
			config.usePushNotifications,
			notifReceiverURL.Hostname(),
			notifReceiverURL.Port(),
			taskID,
			sessionID,
		)
		if err != nil {
			log.Printf("Error completing task: %v", err)
			continue
		}

		if config.history && continueLoop {
			fmt.Println("========= history ======== ")
			taskResponse, err := a2aClient.GetTask(map[string]interface{}{
				"id":            taskID,
				"historyLength": 10,
			})
			if err != nil {
				log.Printf("Error getting task history: %v", err)
				continue
			}

			history := map[string]interface{}{
				"result": map[string]interface{}{
					"history": taskResponse.Result.History,
				},
			}
			jsonBytes, err := json.Marshal(history)
			if err != nil {
				log.Printf("Error marshaling history: %v", err)
				continue
			}
			fmt.Printf("%s\n", string(jsonBytes))
		}
	}
}
