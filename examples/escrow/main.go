package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/fiatjaf/khatru"
	"github.com/fiatjaf/khatru/nip100"
	"github.com/nbd-wtf/go-nostr"
)

func main() {
	relay := khatru.NewRelay()

	// Set up in-memory storage
	store := make(map[string]*nostr.Event)

	// Create validation context with query function
	valCtx := &nip100.ValidationContext{
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

	// Add NIP-100 validation
	relay.RejectEvent = append(relay.RejectEvent,
		func(ctx context.Context, event *nostr.Event) (bool, string) {
			// Debug logging
			fmt.Printf("Validating event kind %d with %d tags\n", event.Kind, len(event.Tags))
			fmt.Printf("Event content: %s\n", event.Content)
			fmt.Printf("Event tags: %+v\n", event.Tags)
			reject, msg := nip100.ValidateEscrowEvent(ctx, event, valCtx)
			if reject {
				fmt.Printf("Event rejected: %s\n", msg)
			}
			return reject, msg
		},
		nip100.PreventFarFutureDeadlines,
	)

	// Add NIP-100 to supported NIPs
	// Add NIP-100 to supported NIPs and ensure we support zaps
	relay.Info.SupportedNIPs = append(relay.Info.SupportedNIPs, 1, 100)

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

	fmt.Println("running escrow relay on :3334")
	http.ListenAndServe(":3334", relay)
}
