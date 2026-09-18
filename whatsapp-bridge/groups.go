package main

// Group management endpoints (added in this fork).
//
//   POST /api/create-group  {"name": "...", "participants": ["15551234567", ...], "topic": "..."}
//   → {"ok": true, "jid": "1203...@g.us"}
//
// Participants are phone numbers with country code or full JIDs. The bridge's
// own number is added by WhatsApp implicitly. Group names are limited to 25
// characters by WhatsApp.

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

type createGroupRequest struct {
	Name         string   `json:"name"`
	Participants []string `json:"participants"`
	Topic        string   `json:"topic"`
}

func registerGroupEndpoints(mux *http.ServeMux, auth func(http.HandlerFunc) http.HandlerFunc, client *whatsmeow.Client) {
	mux.HandleFunc("/api/create-group", auth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req createGroupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || len(req.Participants) == 0 {
			http.Error(w, "name and participants are required", http.StatusBadRequest)
			return
		}
		jids := make([]types.JID, 0, len(req.Participants))
		for _, p := range req.Participants {
			jid, err := resolveRecipientJID(client, p)
			if err != nil {
				http.Error(w, "invalid participant "+p+": "+err.Error(), http.StatusBadRequest)
				return
			}
			jids = append(jids, jid)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		info, err := client.CreateGroup(ctx, whatsmeow.ReqCreateGroup{Name: req.Name, Participants: jids})
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		resp := map[string]interface{}{"ok": true, "jid": info.JID.String()}
		if req.Topic != "" {
			if err := client.SetGroupTopic(ctx, info.JID, "", "", req.Topic); err != nil {
				resp["topic_error"] = err.Error()
			}
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
}
