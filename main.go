package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	client, err := clientFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/invoice", invoiceHandler(client))
	log.Println("invoice service listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func invoiceHandler(client *InfraiClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var order Order
		if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
			http.Error(w, "invalid order", http.StatusBadRequest)
			return
		}
		if order.ID == "" || order.Email == "" || order.Item == "" {
			http.Error(w, "id, email, and item are required", http.StatusBadRequest)
			return
		}
		if err := client.GenerateInvoice(r.Context(), order); err != nil {
			http.Error(w, err.Error(), statusFor(err))
			return
		}
		messageID, err := client.EmailInvoice(r.Context(), order)
		if err != nil {
			http.Error(w, err.Error(), statusFor(err))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(InvoiceResult{MessageID: messageID})
	}
}

func statusFor(err error) int {
	if apiErr, ok := err.(*InfraiError); ok && apiErr.Status >= 400 && apiErr.Status < 500 {
		return apiErr.Status
	}
	return http.StatusBadGateway
}
