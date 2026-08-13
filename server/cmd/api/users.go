package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// Identity is anonymous-first: the app calls POST /users/anonymous once on
// first launch and gets back an id + bearer token — no signup, no email.
// The token is shown exactly once; the DB keeps only its sha256, so a dump
// of the users table impersonates nobody. OAuth linking (user_identities)
// rides on top of this later without changing the user id.

// newToken mints a fresh bearer token and its storable hash. 32 random bytes
// is comfortably beyond guessable; the yk_ prefix makes leaked tokens
// greppable in logs and configs.
func newToken() (token, hash string, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("token entropy: %w", err)
	}
	token = "yk_" + hex.EncodeToString(raw)
	return token, hashToken(token), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

type anonymousUserResponse struct {
	UserID string `json:"user_id"`
	Token  string `json:"token"`
}

func (s *byakuganServer) handleAnonymousUser(w http.ResponseWriter, r *http.Request) {
	token, hash, err := newToken()
	if err != nil {
		log.Printf("[users] token generation failed: %v", err)
		http.Error(w, "could not mint an identity right now", http.StatusServiceUnavailable)
		return
	}

	id, err := s.store.CreateAnonymousUser(r.Context(), hash)
	if err != nil {
		log.Printf("[users] create failed: %v", err)
		http.Error(w, "could not mint an identity right now", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(anonymousUserResponse{UserID: id, Token: token})
}

// userID resolves the request's optional bearer token to a user id.
// Anonymous-first means every failure mode — no header, unknown token, DB
// hiccup — returns "" and the request proceeds unidentified.
func (s *byakuganServer) userID(r *http.Request) string {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || token == "" {
		return ""
	}

	id, err := s.store.UserForToken(r.Context(), hashToken(token))
	if err != nil {
		log.Printf("[users] token resolve failed (continuing anonymous): %v", err)
		return ""
	}
	return id
}
