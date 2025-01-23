package nip100

import (
	"context"
	"encoding/json"
	"time"
  "fmt"

	"github.com/nbd-wtf/go-nostr"
)

// Event kinds for NIP-100
const (
	KindEscrowAgentRegistration = 30400
	KindTaskCreation           = 30401
	KindTaskAcceptance         = 30402
	KindTaskResolution         = 30403
)

// ValidateEscrowEvent validates all NIP-100 events
func ValidateEscrowEvent(ctx context.Context, evt *nostr.Event) (bool, string) {
	switch evt.Kind {
	case KindEscrowAgentRegistration:
		return validateAgentRegistration(evt)
	case KindTaskCreation:
		return validateTaskCreation(evt)
	case KindTaskAcceptance:
		return validateTaskAcceptance(evt)
	case KindTaskResolution:
		return validateTaskResolution(evt)
	}
	return false, ""
}

// PreventFarFutureDeadlines prevents task creation events with deadlines more than 30 days in the future
func PreventFarFutureDeadlines(ctx context.Context, evt *nostr.Event) (bool, string) {
	if evt.Kind != KindTaskCreation {
		return false, ""
	}

	var m struct {
		Deadline int64 `json:"deadline"`
	}
	if err := json.Unmarshal([]byte(evt.Content), &m); err != nil {
		return true, "invalid task creation json"
	}

	maxDeadline := time.Now().AddDate(0, 0, 30).Unix()
	if m.Deadline > maxDeadline {
		return true, "deadline too far in the future (max 30 days)"
	}

	return false, ""
}

func validateAgentRegistration(evt *nostr.Event) (bool, string) {
	var m struct {
		Version                string   `json:"version"`
		PubKey                string   `json:"pubkey"`
		FeeRate               float64  `json:"fee_rate"`
		MinAmount            int      `json:"min_amount"`
		MaxAmount            int      `json:"max_amount"`
		DisputeResolutionPolicy string `json:"dispute_resolution_policy"`
		SupportedCurrencies   []string `json:"supported_currencies"`
	}
	if err := json.Unmarshal([]byte(evt.Content), &m); err != nil {
		return true, "invalid escrow agent registration json"
	}
	if m.Version == "" || m.PubKey == "" || m.FeeRate <= 0 || m.MinAmount <= 0 || m.MaxAmount <= m.MinAmount {
		return true, "invalid escrow agent registration parameters"
	}
	if len(m.SupportedCurrencies) == 0 {
		return true, "must support at least one currency"
	}
	if !evt.Tags.ContainsAny("p", []string{m.PubKey}) {
		return true, "pubkey in content must match p tag"
	}
	return false, ""
}

func validateTaskCreation(evt *nostr.Event) (bool, string) {
	var m struct {
		Version     string `json:"version"`
		TaskID     string `json:"task_id"`
		Description string `json:"description"`
		Amount     int    `json:"amount"`
		PaymentHash string `json:"payment_hash"`
		EscrowAgent string `json:"escrow_agent"`
		Deadline    int64  `json:"deadline"`
		Requirements string `json:"requirements"`
	}
	if err := json.Unmarshal([]byte(evt.Content), &m); err != nil {
		return true, "invalid task creation json"
	}
	if m.Version == "" || m.TaskID == "" || m.Amount <= 0 || m.PaymentHash == "" || m.EscrowAgent == "" {
		return true, "invalid task creation parameters"
	}
	if !evt.Tags.ContainsAny("p", []string{m.EscrowAgent}) {
		return true, "escrow agent in content must match p tag"
	}
	return false, ""
}

func validateTaskAcceptance(evt *nostr.Event) (bool, string) {
	var m struct {
		Version          string `json:"version"`
		TaskID          string `json:"task_id"`
		WorkerCommitment string `json:"worker_commitment"`
	}
	if err := json.Unmarshal([]byte(evt.Content), &m); err != nil {
		return true, "invalid task acceptance json"
	}
	if m.Version == "" || m.TaskID == "" || m.WorkerCommitment == "" {
		return true, "invalid task acceptance parameters"
	}
	// Debug: Print all tags
	fmt.Printf("Task acceptance tags: %+v\n", evt.Tags)
	
	// Check if there's at least one "e" tag
	eRefs := evt.Tags.GetAll([]string{"e"})
	fmt.Printf("Found e tags: %+v\n", eRefs)
	
	if len(eRefs) == 0 {
		return true, "must reference task event"
	}

	// Verify the referenced task event exists and is of kind 30401
	taskEventId := eRefs[0][1]
	fmt.Printf("Task event ID from tag: %s\n", taskEventId)
	
	if taskEventId == "" {
		return true, "invalid task event reference"
	}
	return false, ""
}

func validateTaskResolution(evt *nostr.Event) (bool, string) {
	var m struct {
		Version          string `json:"version"`
		TaskID          string `json:"task_id"`
		Resolution      string `json:"resolution"`
		SettlementProof string `json:"settlement_proof"`
		Details        string `json:"resolution_details"`
	}
	if err := json.Unmarshal([]byte(evt.Content), &m); err != nil {
		return true, "invalid task resolution json"
	}
	if m.Version == "" || m.TaskID == "" || m.Resolution == "" {
		return true, "invalid task resolution parameters"
	}
	if m.Resolution != "settled" && m.Resolution != "disputed" && m.Resolution != "canceled" {
		return true, "invalid resolution status"
	}
	
	eRefs := evt.Tags.GetAll([]string{"e"})
	if len(eRefs) < 2 {
		return true, "must reference both task and acceptance events"
	}
	return false, ""
}
