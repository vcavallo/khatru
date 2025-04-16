package nip3400

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
)

// Event kinds for Catallax NIP-3400
const (
	KindArbiterAnnouncement = 33400
	KindTaskProposal       = 33401
	KindTaskConclusion     = 3402
)

// ValidationContext holds functions needed for validation
type ValidationContext struct {
	QueryEvents func(context.Context, nostr.Filter) (chan *nostr.Event, error)
}

// ValidateEscrowEvent validates all NIP-3400 events
func ValidateEscrowEvent(ctx context.Context, evt *nostr.Event, valCtx *ValidationContext) (bool, string) {
	switch evt.Kind {
	case KindArbiterAnnouncement:
		return validateArbiterAnnouncement(evt)
	case KindTaskProposal:
		return validateTaskProposal(evt)
	case KindTaskConclusion:
		return validateTaskConclusion(ctx, evt, valCtx)
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
		return true, "invalid task proposal json"
	}

	// If deadline is optional and not specified, don't reject
	if m.Deadline == 0 {
		return false, ""
	}

	// Calculate 30 days from now in Unix timestamp
	maxDeadline := nostr.Now() + 30*24*60*60 // 30 days in seconds
	if m.Deadline > int64(maxDeadline) {
		return true, "deadline too far in the future (max 30 days)"
	}

	return false, ""
}

func validateArbiterAnnouncement(evt *nostr.Event) (bool, string) {
	fmt.Printf("Validating arbiter announcement. Content: %s\n", evt.Content)
	fmt.Printf("All tags: %+v\n", evt.Tags)
	
	var m struct {
		Name        string `json:"name"`
		About       string `json:"about"`
		PolicyText  string `json:"policy_text"`
		PolicyURL   string `json:"policy_url"`
	}
	if err := json.Unmarshal([]byte(evt.Content), &m); err != nil {
		return true, "invalid arbiter announcement json"
	}
	if m.Name == "" {
		return true, "name is required in arbiter announcement"
	}
	
	// Check for required d tag (service identifier)
	dTags := evt.Tags.GetAll([]string{"d"})
	if len(dTags) == 0 {
		return true, "must include d tag with service identifier"
	}
	
	// Verify arbiter pubkey matches event pubkey
	if !evt.Tags.ContainsAny("p", []string{evt.PubKey}) {
		return true, "must include arbiter pubkey in p tag"
	}
	
	// Check fee type and amount
	feeTypeTags := evt.Tags.GetAll([]string{"fee_type"})
	if len(feeTypeTags) == 0 {
		return true, "must include fee_type tag"
	}
	if len(feeTypeTags[0]) < 2 {
		return true, "invalid fee_type tag format"
	}
	feeType := feeTypeTags[0][1]
	if feeType != "flat" && feeType != "percentage" {
		return true, "fee_type must be either 'flat' or 'percentage'"
	}
	
	feeAmountTags := evt.Tags.GetAll([]string{"fee_amount"})
	if len(feeAmountTags) == 0 {
		return true, "must include fee_amount tag"
	}
	if len(feeAmountTags[0]) < 2 {
		return true, "invalid fee_amount tag format"
	}
	
	return false, ""
}

func validateTaskProposal(evt *nostr.Event) (bool, string) {
	fmt.Printf("Validating task proposal. Content: %s\n", evt.Content)
	fmt.Printf("All tags: %+v\n", evt.Tags)
	
	var m struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Requirements string `json:"requirements"`
		Deadline    int64  `json:"deadline"`
	}
	if err := json.Unmarshal([]byte(evt.Content), &m); err != nil {
		return true, "invalid task proposal json"
	}
	if m.Title == "" || m.Description == "" || m.Requirements == "" {
		return true, "title, description, and requirements are required in task proposal"
	}
	
	// Check for required d tag (task identifier)
	dTags := evt.Tags.GetAll([]string{"d"})
	if len(dTags) == 0 {
		return true, "must include d tag with task identifier"
	}
	
	// Check for required tags
	pTags := evt.Tags.GetAll([]string{"p"})
	if len(pTags) < 2 {
		return true, "must include patron and arbiter pubkeys in p tags"
	}
	
	// Verify patron pubkey matches event pubkey
	patronFound := false
	for _, tag := range pTags {
		if len(tag) > 1 && tag[1] == evt.PubKey {
			patronFound = true
			break
		}
	}
	if !patronFound {
		return true, "patron pubkey must match event pubkey"
	}
	
	// Check for arbiter service reference in a tag
	aTags := evt.Tags.GetAll([]string{"a"})
	if len(aTags) == 0 {
		return true, "must include a tag referencing arbiter service"
	}
	
	// Check amount tag
	amountTags := evt.Tags.GetAll([]string{"amount"})
	if len(amountTags) == 0 {
		return true, "must include amount tag"
	}
	if len(amountTags[0]) < 2 {
		return true, "invalid amount tag format"
	}
	
	// Check status tag
	statusTags := evt.Tags.GetAll([]string{"status"})
	if len(statusTags) == 0 {
		return true, "must include status tag"
	}
	if len(statusTags[0]) < 2 {
		return true, "invalid status tag format"
	}
	
	status := statusTags[0][1]
	validStatuses := []string{"proposed", "funded", "in_progress", "submitted", "concluded"}
	statusValid := false
	for _, validStatus := range validStatuses {
		if status == validStatus {
			statusValid = true
			break
		}
	}
	if !statusValid {
		return true, "invalid status value"
	}
	
	// If status is funded or later, check for zap receipt
	if status == "funded" || status == "in_progress" || status == "submitted" || status == "concluded" {
		zapTags := evt.Tags.GetAll([]string{"e"})
		zapFound := false
		for _, tag := range zapTags {
			if len(tag) > 2 && tag[2] == "zap" {
				zapFound = true
				break
			}
		}
		if !zapFound {
			return true, "funded status requires zap receipt reference"
		}
	}
	
	// If status is in_progress or later, check for worker pubkey
	if status == "in_progress" || status == "submitted" || status == "concluded" {
		if len(pTags) < 3 {
			return true, "in_progress status requires worker pubkey in p tag"
		}
	}
	
	return false, ""
}

func validateTaskConclusion(ctx context.Context, evt *nostr.Event, valCtx *ValidationContext) (bool, string) {
	fmt.Printf("Validating task conclusion. Content: %s\n", evt.Content)
	fmt.Printf("All tags: %+v\n", evt.Tags)
	
	var m struct {
		ResolutionDetails string `json:"resolution_details"`
	}
	if err := json.Unmarshal([]byte(evt.Content), &m); err != nil {
		return true, "invalid task conclusion json"
	}
	if m.ResolutionDetails == "" {
		return true, "resolution_details is required in task conclusion"
	}
	
	// Must include resolution tag
	resolutionTags := evt.Tags.GetAll([]string{"resolution"})
	if len(resolutionTags) == 0 {
		return true, "must include resolution tag"
	}
	if len(resolutionTags[0]) < 2 {
		return true, "invalid resolution tag format"
	}
	
	resolution := resolutionTags[0][1]
	validResolutions := []string{"successful", "rejected", "cancelled", "abandoned"}
	resolutionValid := false
	for _, validResolution := range validResolutions {
		if resolution == validResolution {
			resolutionValid = true
			break
		}
	}
	if !resolutionValid {
		return true, "invalid resolution value"
	}
	
	// Must include all three participant pubkeys
	pTags := evt.Tags.GetAll([]string{"p"})
	if len(pTags) < 3 {
		return true, "must include patron, arbiter, and worker pubkeys in p tags"
	}
	
	// Verify arbiter pubkey matches event pubkey
	arbiterFound := false
	for _, tag := range pTags {
		if len(tag) > 1 && tag[1] == evt.PubKey {
			arbiterFound = true
			break
		}
	}
	if !arbiterFound {
		return true, "arbiter pubkey must match event pubkey"
	}
	
	// Must reference task proposal and payout zap receipt events
	eRefs := evt.Tags.GetAll([]string{"e"})
	if len(eRefs) < 2 {
		return true, "must reference both task proposal and payout zap receipt events"
	}
	
	// Must include addressable reference to task proposal
	aTags := evt.Tags.GetAll([]string{"a"})
	if len(aTags) == 0 {
		return true, "must include a tag referencing task proposal"
	}
	
	return false, ""
}