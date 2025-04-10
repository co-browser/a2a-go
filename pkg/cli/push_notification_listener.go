package cli

import (
	"a2a-go/pkg/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
)

// PushNotificationListener handles incoming push notifications from the agent
type PushNotificationListener struct {
	host                   string
	port                   string
	notificationReceiverAuth *utils.PushNotificationReceiverAuth
	server                 *http.Server
	wg                     sync.WaitGroup
}

// NewPushNotificationListener creates a new push notification listener
func NewPushNotificationListener(host, port string, auth *utils.PushNotificationReceiverAuth) *PushNotificationListener {
	return &PushNotificationListener{
		host:                   host,
		port:                   port,
		notificationReceiverAuth: auth,
	}
}

// Start starts the push notification listener server
func (l *PushNotificationListener) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/notify", l.handleNotification)
	mux.HandleFunc("/.well-known/validation-token", l.handleValidationCheck)

	l.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%s", l.host, l.port),
		Handler: mux,
	}

	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		log.Printf("Starting push notification listener on %s:%s", l.host, l.port)
		if err := l.server.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("Push notification listener error: %v", err)
		}
	}()
}

// Stop stops the push notification listener server
func (l *PushNotificationListener) Stop() {
	if l.server != nil {
		if err := l.server.Close(); err != nil {
			log.Printf("Error closing push notification listener: %v", err)
		}
		l.wg.Wait()
	}
}

func (l *PushNotificationListener) handleValidationCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := r.Header.Get("Validation-Token")
	if token == "" {
		http.Error(w, "Validation token not found", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(token))
}

func (l *PushNotificationListener) handleNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify JWT token
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
		return
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")

	if err := l.notificationReceiverAuth.VerifyToken(token); err != nil {
		http.Error(w, fmt.Sprintf("Invalid token: %v", err), http.StatusUnauthorized)
		return
	}

	var notification map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		http.Error(w, fmt.Sprintf("Error decoding notification: %v", err), http.StatusBadRequest)
		return
	}

	log.Printf("Received notification: %+v", notification)
	w.WriteHeader(http.StatusOK)
} 