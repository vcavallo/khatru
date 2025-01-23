package nip100

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
)

// Event kinds for NIP-100
const (
	KindEscrowAgentRegistration = 3400
	KindTaskProposal           = 3401
	KindAgentTaskAcceptance    = 3402
	KindTaskFinalization       = 3403
	KindWorkerApplication      = 3404
	KindWorkerAssignment       = 3405
	KindWorkSubmission         = 3406
	KindTaskResolution         = 3407
)

// ValidationContext holds functions needed for validation
type ValidationContext struct {
	QueryEvents func(context.Context, nostr.Filter) (chan *nostr.Event, error)
}

// ValidateEscrowEvent validates all NIP-100 events
func ValidateEscrowEvent(ctx context.Context, evt *nostr.Event, valCtx *ValidationContext) (bool, string) {
	switch evt.Kind {
	case KindEscrowAgentRegistration:
		return validateAgentRegistration(evt)
	case KindTaskProposal:
		return validateTaskProposal(evt)
	case KindAgentTaskAcceptance:
		return validateAgentTaskAcceptance(ctx, evt, valCtx)
	case KindTaskFinalization:
		return validateTaskFinalization(ctx, evt, valCtx)
	case KindWorkerApplication:
		return validateWorkerApplication(ctx, evt, valCtx)
	case KindWorkerAssignment:
		return validateWorkerAssignment(ctx, evt, valCtx)
	case KindWorkSubmission:
		return validateWorkSubmission(ctx, evt, valCtx)
	case KindTaskResolution:
		return validateTaskResolution(ctx, evt, valCtx)
	}
	return false, ""
}

// PreventFarFutureDeadlines prevents task creation events with deadlines more than 30 days in the future
func PreventFarFutureDeadlines(ctx context.Context, evt *nostr.Event) (bool, string) {
	if evt.Kind != KindTaskProposal {
		return false, ""
	}

	var m struct {
		Deadline int64 `json:"deadline"`
	}
	if err := json.Unmarshal([]byte(evt.Content), &m); err != nil {
		return true, "invalid task creation json"
	}

	// Calculate 30 days from now in Unix timestamp
	maxDeadline := nostr.Now() + 30*24*60*60 // 30 days in seconds
	if m.Deadline > int64(maxDeadline) {
		return true, "deadline too far in the future (max 30 days)"
	}

	return false, ""
}

func validateAgentRegistration(evt *nostr.Event) (bool, string) {
	var m struct {
		Name                  string   `json:"name"`
		About                 string   `json:"about"`
		FeeRate               float64  `json:"fee_rate"`
		MinAmount            int      `json:"min_amount"`
		MaxAmount            int      `json:"max_amount"`
		DisputeResolutionPolicy string `json:"dispute_resolution_policy"`
		SupportedCurrencies   []string `json:"supported_currencies"`
	}
	if err := json.Unmarshal([]byte(evt.Content), &m); err != nil {
		return true, "invalid escrow agent registration json"
	}
	if m.Name == "" || m.About == "" || m.FeeRate <= 0 || m.MinAmount <= 0 || m.MaxAmount <= m.MinAmount {
		return true, "invalid escrow agent registration parameters"
	}
	if len(m.SupportedCurrencies) == 0 {
		return true, "must support at least one currency"
	}
	if !evt.Tags.ContainsAny("p", []string{evt.PubKey}) {
		return true, "must include agent pubkey in p tag"
	}
	return false, ""
}

func validateTaskProposal(evt *nostr.Event) (bool, string) {
	fmt.Printf("Validating task proposal. Content: %s\n", evt.Content)
	fmt.Printf("All tags: %+v\n", evt.Tags)
	
	amountTags := evt.Tags.GetAll([]string{"amount"})
	fmt.Printf("Amount tags found: %+v\n", amountTags)
	var m struct {
		Description  string `json:"description"`
		Requirements string `json:"requirements"`
		Deadline     int64  `json:"deadline"`
	}
	if err := json.Unmarshal([]byte(evt.Content), &m); err != nil {
		return true, "invalid task proposal json"
	}
	if m.Description == "" || m.Requirements == "" || m.Deadline <= 0 {
		return true, "invalid task proposal parameters"
	}
	
	// Check for required tags - need both agent and creator pubkeys
	pTags := evt.Tags.GetAll([]string{"p"})
	if len(pTags) < 2 {
		return true, "must include both agent and creator pubkeys in p tags"
	}
	
	// Verify creator pubkey matches event pubkey
	creatorFound := false
	for _, tag := range pTags {
		if len(tag) > 1 && tag[1] == evt.PubKey {
			creatorFound = true
			break
		}
	}
	if !creatorFound {
		return true, "creator pubkey must match event pubkey"
	}
	
	// Check amount tag
	amountTags = evt.Tags.GetAll([]string{"amount"})
	if len(amountTags) == 0 {
		return true, "must include amount tag"
	}
	if len(amountTags[0]) < 2 {
		return true, "invalid amount tag format"
	}

	return false, ""
}

func validateAgentTaskAcceptance(ctx context.Context, evt *nostr.Event, valCtx *ValidationContext) (bool, string) {
	// Must reference task proposal event
	eRefs := evt.Tags.GetAll([]string{"e"})
	if len(eRefs) == 0 {
		return true, "must reference task proposal event"
	}

	// Must include both creator and agent pubkeys
	pTags := evt.Tags.GetAll([]string{"p"})
	if len(pTags) < 2 {
		return true, "must include both creator and agent pubkeys in p tags"
	}

	// Verify agent pubkey matches event pubkey
	agentFound := false
	for _, tag := range pTags {
		if len(tag) > 1 && tag[1] == evt.PubKey {
			agentFound = true
			break
		}
	}
	if !agentFound {
		return true, "agent pubkey must match event pubkey"
	}

	// Check if task has already been accepted
	taskEventId := eRefs[0][1]
	acceptanceFilter := nostr.Filter{
		Kinds: []int{KindAgentTaskAcceptance},
		Tags: nostr.TagMap{
			"e": []string{taskEventId},
		},
	}
	
	acceptanceEvents, err := valCtx.QueryEvents(ctx, acceptanceFilter)
	if err == nil {
		for evt := range acceptanceEvents {
			if evt != nil {
				return true, "task has already been accepted"
			}
		}
	}

	return false, ""
}

func validateTaskFinalization(ctx context.Context, evt *nostr.Event, valCtx *ValidationContext) (bool, string) {
	// Must reference agent acceptance and zap receipt events
	eRefs := evt.Tags.GetAll([]string{"e"})
	if len(eRefs) < 2 {
		return true, "must reference both acceptance and zap receipt events"
	}

	// Must include both creator and agent pubkeys
	pTags := evt.Tags.GetAll([]string{"p"})
	if len(pTags) < 2 {
		return true, "must include both creator and agent pubkeys in p tags"
	}

	// Verify creator pubkey matches event pubkey
	creatorFound := false
	for _, tag := range pTags {
		if len(tag) > 1 && tag[1] == evt.PubKey {
			creatorFound = true
			break
		}
	}
	if !creatorFound {
		return true, "creator pubkey must match event pubkey"
	}

	// Check amount tag
	amountTags := evt.Tags.GetAll([]string{"amount"})
	if len(amountTags) == 0 {
		return true, "must include amount tag"
	}
	if len(amountTags[0]) < 2 {
		return true, "invalid amount tag format"
	}

	return false, ""
}

func validateWorkerApplication(ctx context.Context, evt *nostr.Event, valCtx *ValidationContext) (bool, string) {
	if evt.Content == "" {
		return true, "must include application details in content"
	}

	// Must reference finalized task event
	eRefs := evt.Tags.GetAll([]string{"e"})
	if len(eRefs) == 0 {
		return true, "must reference finalized task event"
	}

	// Must include creator and agent pubkeys
	if len(evt.Tags.GetAll([]string{"p"})) < 2 {
		return true, "must include creator and agent pubkeys in p tags"
	}

	return false, ""
}

func validateWorkerAssignment(ctx context.Context, evt *nostr.Event, valCtx *ValidationContext) (bool, string) {
	// Must reference finalized task and worker application events
	eRefs := evt.Tags.GetAll([]string{"e"})
	if len(eRefs) < 2 {
		return true, "must reference both task and application events"
	}

	// Must include worker and agent pubkeys
	if len(evt.Tags.GetAll([]string{"p"})) < 2 {
		return true, "must include worker and agent pubkeys in p tags"
	}

	return false, ""
}

func validateWorkSubmission(ctx context.Context, evt *nostr.Event, valCtx *ValidationContext) (bool, string) {
	if evt.Content == "" {
		return true, "must include work details/proof in content"
	}

	// Must reference worker assignment event
	eRefs := evt.Tags.GetAll([]string{"e"})
	if len(eRefs) == 0 {
		return true, "must reference worker assignment event"
	}

	// Must include creator and agent pubkeys
	if len(evt.Tags.GetAll([]string{"p"})) < 2 {
		return true, "must include creator and agent pubkeys in p tags"
	}

	return false, ""
}

func validateTaskResolution(ctx context.Context, evt *nostr.Event, valCtx *ValidationContext) (bool, string) {
	var m struct {
		Resolution       string `json:"resolution"`
		ResolutionDetails string `json:"resolution_details"`
	}
	if err := json.Unmarshal([]byte(evt.Content), &m); err != nil {
		return true, "invalid task resolution json"
	}
	if m.Resolution == "" || m.ResolutionDetails == "" {
		return true, "invalid task resolution parameters"
	}
	if m.Resolution != "completed" && m.Resolution != "rejected" && m.Resolution != "canceled" {
		return true, "invalid resolution status"
	}
	
	// Must reference work submission and zap receipt events
	eRefs := evt.Tags.GetAll([]string{"e"})
	if len(eRefs) < 2 {
		return true, "must reference work submission and zap receipt events"
	}

	// Must include creator and worker pubkeys and amount
	if len(evt.Tags.GetAll([]string{"p"})) < 2 {
		return true, "must include creator and worker pubkeys in p tags"
	}
	amountTags := evt.Tags.GetAll([]string{"amount"})
	if len(amountTags) == 0 {
		return true, "must include amount tag"
	}
	if len(amountTags[0]) < 2 {
		return true, "invalid amount tag format"
	}
	return false, ""
}
