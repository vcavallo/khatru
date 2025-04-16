# Testing NIP-3400 Catallax Implementation

This guide walks through testing the complete Catallax workflow using the `nak` CLI tool.

## Implementation Details

The relay implements NIP-3400 (Catallax) with the following validations:

### Event Kinds
- 33400: Arbiter Announcement - Parameterized replaceable event for arbiter service details
- 33401: Task Proposal - Parameterized replaceable event that tracks task status through its lifecycle
- 3402: Task Conclusion - Regular event that documents the final resolution of a task

### Validation Rules
- All events must have proper pubkey tags matching the event signer
- Amount tags are required for task proposals and conclusions
- Task proposals must include a status tag
- Task deadlines cannot be more than 30 days in the future
- Task resolution must be "successful", "rejected", "cancelled", or "abandoned"
- Events are validated for proper sequence and references

### Required Tags
- Arbiter Announcement: d (service identifier), p (arbiter pubkey), fee_type, fee_amount
- Task Proposal: d (task identifier), p (pubkeys), a (arbiter reference), amount, status
- Task Conclusion: e (references), p (pubkeys), resolution, a (task reference)

## Notes

1. All commands assume the relay is running at ws://localhost:3334
2. The payment_hash would come from a real Lightning Network invoice
3. Real implementations should use persistent storage
4. Error handling is minimal in these examples

# Testing / Example

**you must have `jq` installed; sorry**.

## Start the Relay

```bash
# Build and run the relay
# From the root of the khatru repo:
go build -o escrow-relay examples/escrow/main.go
./escrow-relay
```

The relay will be available at `ws://localhost:3334`. When using the `nak` tool, make sure to include the WebSocket protocol:

```bash
# Correct format:
nak event --content "hello" ws://localhost:3334

# Incorrect format:
nak event "hello" localhost:3334  # This will fail
```

This repo provides `examples/escrow/test.sh` which runs a 'happy-path' test case.

## Complete Workflow

The test script demonstrates the complete workflow:

1. Arbiter announces their service (kind 33400)
2. Patron creates a task proposal (kind 33401 with status "proposed")
3. Patron funds the task and updates it (kind 33401 with status "funded")
4. Patron assigns a worker and updates the task (kind 33401 with status "in_progress")
5. Patron marks work as submitted (kind 33401 with status "submitted")
6. Arbiter concludes the task (kind 3402)

Run the test script to see the entire workflow:

```bash
./examples/escrow/test.sh
```