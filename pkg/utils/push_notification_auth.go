// PushNotificationAuth.go: Go version of the push notification auth system

package utils

import (
	"bytes"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const AuthHeaderPrefix = "Bearer "

type PushNotificationAuth struct{}

func (p *PushNotificationAuth) calculateRequestBodySHA256(data map[string]interface{}) (string, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(data)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(buf.Bytes())
	return fmt.Sprintf("%x", hash[:]), nil
}

type PushNotificationSenderAuth struct {
	PushNotificationAuth
	privateKey *rsa.PrivateKey
	publicKeys []map[string]interface{}
	lock       sync.Mutex
}

func (s *PushNotificationSenderAuth) VerifyPushNotificationURL(url string) bool {
	validationToken := uuid.NewString()
	resp, err := http.Get(fmt.Sprintf("%s?validationToken=%s", url, validationToken))
	if err != nil {
		log.Printf("Verification error: %v", err)
		return false
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	verified := string(body) == validationToken
	log.Printf("Verified push-notification URL: %s => %v", url, verified)
	return verified
}

func (s *PushNotificationSenderAuth) GenerateRSAKey() error {
	privateKey, err := rsa.GenerateKey(nil, 2048)
	if err != nil {
		return err
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	s.privateKey = privateKey

	// Simplified public JWK export
	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return err
	}
	pemPub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
	s.publicKeys = append(s.publicKeys, map[string]interface{}{
		"alg": "RS256",
		"use": "sig",
		"kid": uuid.NewString(),
		"pem": string(pemPub),
	})
	return nil
}

func (s *PushNotificationSenderAuth) GenerateJWT(data map[string]interface{}) (string, error) {
	shaDigest, err := s.calculateRequestBodySHA256(data)
	if err != nil {
		return "", err
	}
	claims := jwt.MapClaims{
		"iat":                time.Now().Unix(),
		"request_body_sha256": shaDigest,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "example-kid" // Dummy kid

	return token.SignedString(s.privateKey)
}

func (s *PushNotificationSenderAuth) SendPushNotification(url string, data map[string]interface{}) error {
	token, err := s.GenerateJWT(data)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Push-notification failed: %v", err)
		return err
	}
	defer resp.Body.Close()
	log.Printf("Push-notification sent to %s", url)
	return nil
}

type PushNotificationReceiverAuth struct {
	PushNotificationAuth
	publicKey *rsa.PublicKey
}

func (r *PushNotificationReceiverAuth) LoadJWKS(pemData string) error {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil || block.Type != "PUBLIC KEY" {
		return errors.New("invalid PEM data")
	}
	parsedKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return err
	}
	if pub, ok := parsedKey.(*rsa.PublicKey); ok {
		r.publicKey = pub
		return nil
	}
	return errors.New("not an RSA public key")
}

func (r *PushNotificationReceiverAuth) VerifyPushNotification(req *http.Request) (bool, error) {
	authHeader := req.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, AuthHeaderPrefix) {
		return false, errors.New("invalid auth header")
	}
	tokenStr := strings.TrimPrefix(authHeader, AuthHeaderPrefix)

	parsedToken, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return r.publicKey, nil
	})
	if err != nil || !parsedToken.Valid {
		return false, err
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok || !parsedToken.Valid {
		return false, errors.New("invalid token claims")
	}

	if time.Since(time.Unix(int64(claims["iat"].(float64)), 0)) > 5*time.Minute {
		return false, errors.New("token expired")
	}

	var requestBody map[string]interface{}
	bodyBytes, _ := io.ReadAll(req.Body)
	_ = json.Unmarshal(bodyBytes, &requestBody)

	actualDigest, _ := r.calculateRequestBodySHA256(requestBody)
	if actualDigest != claims["request_body_sha256"] {
		return false, errors.New("digest mismatch")
	}

	return true, nil
}

func (r *PushNotificationReceiverAuth) VerifyToken(tokenStr string) error {
	parsedToken, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return r.publicKey, nil
	})
	if err != nil || !parsedToken.Valid {
		return err
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok || !parsedToken.Valid {
		return errors.New("invalid token claims")
	}

	if time.Since(time.Unix(int64(claims["iat"].(float64)), 0)) > 5*time.Minute {
		return errors.New("token expired")
	}

	return nil
}
