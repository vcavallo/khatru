package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/fiatjaf/khatru"
	"github.com/fiatjaf/khatru/nip3400"
	"github.com/nbd-wtf/go-nostr"
)

func main() {
	relay := khatru.NewRelay()

	// Set up in-memory storage
	store := make(map[string]*nostr.Event)

	// Create validation context with query function
	valCtx := &nip3400.ValidationContext{
		QueryEvents: func(ctx context.Context, filter nostr.Filter) (chan *nostr.Event, error) {
			ch := make(chan *nostr.Event)
			go func() {
				for _, evt := range store {
					if filter.Matches(evt) {
						ch <- evt
					}
				}
				close(ch)
			}()
			return ch, nil
		},
	}

	// Add NIP-3400 validation
	relay.RejectEvent = append(relay.RejectEvent,
		func(ctx context.Context, event *nostr.Event) (bool, string) {
			// Allow regular text notes to pass through
			if event.Kind == 1 {
				return false, ""
			}

			// Debug logging
			fmt.Printf("Validating event kind %d with %d tags\n", event.Kind, len(event.Tags))
			fmt.Printf("Event content: %s\n", event.Content)
			fmt.Printf("Event tags: %+v\n", event.Tags)
			
			// Check if this is a Catallax event
			isCatallaxEvent := event.Kind == nip3400.KindArbiterAnnouncement ||
				event.Kind == nip3400.KindTaskProposal ||
				event.Kind == nip3400.KindTaskConclusion
			
			if isCatallaxEvent {
				reject, msg := nip3400.ValidateEscrowEvent(ctx, event, valCtx)
				if reject {
					fmt.Printf("Event rejected: %s\n", msg)
				}
				return reject, msg
			}
			
			return false, ""
		},
		nip3400.PreventFarFutureDeadlines,
	)

	// Add NIP-3400 to supported NIPs
	// Also support NIP-1 (basic protocol) and NIP-57 for zaps
	relay.Info.SupportedNIPs = append(relay.Info.SupportedNIPs, 1, 57, 33400)

	// Add storage handlers
	relay.StoreEvent = append(relay.StoreEvent,
		func(ctx context.Context, event *nostr.Event) error {
			store[event.ID] = event
			return nil
		},
	)

	relay.QueryEvents = append(relay.QueryEvents,
		func(ctx context.Context, filter nostr.Filter) (chan *nostr.Event, error) {
			ch := make(chan *nostr.Event)
			go func() {
				for _, evt := range store {
					if filter.Matches(evt) {
						ch <- evt
					}
				}
				close(ch)
			}()
			return ch, nil
		},
	)

	fmt.Println("running Catallax relay on ws://localhost:3334")
	http.ListenAndServe(":3334", relay)
}