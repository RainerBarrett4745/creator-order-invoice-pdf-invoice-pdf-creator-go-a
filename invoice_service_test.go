package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOrderDecisionGeneratesThenEmails(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if r.URL.Path == "/pdf/generate" && body["html"] == nil {
			t.Fatal("invoice HTML is required")
		}
		if r.URL.Path == "/email/send" && body["html"] == nil {
			t.Fatal("subscriber HTML is required")
		}
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "data": map[string]any{"message_id": "msg-42"}})
	}))
	defer server.Close()
	client := NewInfraiClient("test-key")
	client.baseURL = server.URL
	order := Order{ID: "ord-7", Creator: "Mina", Subscriber: "Reader", Email: "reader@example.com", Item: "Studio pack", Amount: "25.00"}
	if err := client.GenerateInvoice(context.Background(), order); err != nil {
		t.Fatal(err)
	}
	messageID, err := client.EmailInvoice(context.Background(), order)
	if err != nil {
		t.Fatal(err)
	}
	if messageID != "msg-42" {
		t.Fatalf("message id = %q", messageID)
	}
	if got := len(paths); got != 2 || paths[0] != "POST /pdf/generate" || paths[1] != "POST /email/send" {
		t.Fatalf("calls = %#v", paths)
	}
}
