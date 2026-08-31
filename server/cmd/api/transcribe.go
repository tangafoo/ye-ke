package main

import (
	"encoding/json"
	"log"
	"net/http"
)

const maxVoiceNoteBytes = 5 << 20

type transcribeResponse struct {
	Text         string `json:"text"`
	LanguageCode string `json:"language_code"`
}

func (s *byakuganServer) handleTranscribe(w http.ResponseWriter, r *http.Request) {
	// me := s.userID(r)

	r.Body = http.MaxBytesReader(w, r.Body, maxVoiceNoteBytes)

	file, header, err := r.FormFile("audio")
	if err != nil {
		http.Error(w, "expected multipart field 'audio' under 5MB", http.StatusBadRequest)
		return
	}
	defer file.Close()

	t, err := s.scribe.Transcribe(r.Context(), header.Filename, file)
	if err != nil {
		log.Printf("[transcribe] %v", err)
		http.Error(w, "having trouble with our transcribing engine. please try again later", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transcribeResponse{
		Text:         t.Text,
		LanguageCode: t.LanguageCode,
	})

}
