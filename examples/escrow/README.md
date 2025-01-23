# Testing NIP-100 Escrow Implementation

This guide walks through testing the complete escrow workflow using `nak` CLI tool.

**you must have `jq` installed; sorry**.

## Start the Relay

```bash
# Build and run the relay
go build -o escrow-relay main.go
./escrow-relay
```

## Generate Test Keys

```bash
# Generate keys for each party (save these for the test session)
echo "Generating escrow agent key..."
AGENT_KEY=$(nak key generate)
AGENT_PUB=$(nak key public $AGENT_KEY)
echo "Agent pubkey: $AGENT_PUB"

echo "Generating task creator key..."
CREATOR_KEY=$(nak key generate)
CREATOR_PUB=$(nak key public $CREATOR_KEY)
echo "Creator pubkey: $CREATOR_PUB"

echo "Generating worker key..."
WORKER_KEY=$(nak key generate)
WORKER_PUB=$(nak key public $WORKER_KEY)
echo "Worker pubkey: $WORKER_PUB"
```

## Test Complete Workflow

Event kinds used in this workflow:
- 30400: Escrow Agent Registration
- 30401: Task Creation
- 30402: Task Acceptance
- 30403: Task Resolution (final settlement)

### 1. Register Escrow Agent

```bash
# Register the escrow agent
AGENT_EVENT=$(nak event --sec $AGENT_KEY -k 30400 -c "{
  \"version\": \"1.0.0\",
  \"pubkey\": \"$AGENT_PUB\",
  \"fee_rate\": 0.01,
  \"min_amount\": 1000,
  \"max_amount\": 1000000,
  \"dispute_resolution_policy\": \"Mediation first, then arbitration\",
  \"supported_currencies\": [\"BTC\"]
}" -t p=$AGENT_PUB -t t=ln-escrow-agent ws://localhost:3334)

# Save the event ID
AGENT_EVENT_ID=$(echo $AGENT_EVENT | jq -r .id)
echo "Agent registration event ID: $AGENT_EVENT_ID"
```

### 2. Create Task

```bash
# Create a task (deadline 7 days from now)
DEADLINE=$(date -d "+7 days" +%s)
TASK_EVENT=$(nak event --sec $CREATOR_KEY -k 30401 -c "{
  \"version\": \"1.0.0\",
  \"task_id\": \"$(uuidgen)\",
  \"description\": \"Create a nostr client\",
  \"amount\": 100000,
  \"payment_hash\": \"hash123\",
  \"escrow_agent\": \"$AGENT_PUB\",
  \"deadline\": $DEADLINE,
  \"requirements\": \"Must support NIPs 1,2,4\"
}" -t p=$AGENT_PUB -t amount=100000,sat ws://localhost:3334)

# Save the event ID
TASK_EVENT_ID=$(echo $TASK_EVENT | jq -r .id)
echo "Task creation event ID: $TASK_EVENT_ID"
```

### 3. Accept Task

```bash
# Worker accepts the task
ACCEPT_EVENT=$(nak event --sec $WORKER_KEY -k 30402 -c "{
  \"version\": \"1.0.0\",
  \"task_id\": \"$TASK_EVENT_ID\",
  \"worker_commitment\": \"I agree to complete this task according to requirements\"
}" -t e=$TASK_EVENT_ID -t p=$CREATOR_PUB -t p=$WORKER_PUB -t p=$AGENT_PUB ws://localhost:3334)

# Save the event ID
ACCEPT_EVENT_ID=$(echo $ACCEPT_EVENT | jq -r .id)
echo "Task acceptance event ID: $ACCEPT_EVENT_ID"
```

### 4. Resolve Task

```bash
# Resolve the task (can be done by agent)
nak event --sec $AGENT_KEY -k 30403 -c "{
  \"version\": \"1.0.0\",
  \"task_id\": \"$TASK_EVENT_ID\",
  \"resolution\": \"settled\",
  \"settlement_proof\": \"lightning_txid_123\",
  \"resolution_details\": \"Task completed successfully\"
}" -t e=$TASK_EVENT_ID -t e=$ACCEPT_EVENT_ID -t p=$CREATOR_PUB -t p=$WORKER_PUB -t p=$AGENT_PUB ws://localhost:3334
```

## Query Events

```bash
# Query all escrow-related events
echo "All escrow agent registrations:"
nak req -k 30400 ws://localhost:3334

echo "All tasks:"
nak req -k 30401 ws://localhost:3334

echo "All task acceptances:"
nak req -k 30402 ws://localhost:3334

echo "All task resolutions:"
nak req -k 30403 ws://localhost:3334

# Query specific task thread
echo "Complete thread for task $TASK_EVENT_ID:"
nak req --ids $TASK_EVENT_ID,$ACCEPT_EVENT_ID ws://localhost:3334
```

## Validation Tests

```bash
# Test invalid deadline (too far in future)
LONG_DEADLINE=$(date -d "+60 days" +%s)
echo "Testing invalid deadline..."
nak event --sec $CREATOR_KEY -k 30401 -c "{
  \"version\": \"1.0.0\",
  \"task_id\": \"$(uuidgen)\",
  \"description\": \"Test task\",
  \"amount\": 100000,
  \"payment_hash\": \"hash123\",
  \"escrow_agent\": \"$AGENT_PUB\",
  \"deadline\": $LONG_DEADLINE,
  \"requirements\": \"Test requirements\"
}" -t p=$AGENT_PUB -t amount=100000,sat ws://localhost:3334

# Test invalid resolution status
echo "Testing invalid resolution status..."
nak event --sec $AGENT_KEY -k 30403 -c "{
  \"version\": \"1.0.0\",
  \"task_id\": \"$TASK_EVENT_ID\",
  \"resolution\": \"invalid_status\",
  \"settlement_proof\": \"proof123\",
  \"resolution_details\": \"Invalid resolution\"
}" -t e=$TASK_EVENT_ID -t e=$ACCEPT_EVENT_ID -t p=$CREATOR_PUB -t p=$WORKER_PUB -t p=$AGENT_PUB ws://localhost:3334
```

## Clean Up

The relay uses in-memory storage, so simply stopping and restarting the relay will clear all events.

## Notes

1. All commands assume the relay is running at ws://localhost:3334
2. The UUIDs are generated randomly - in practice you might want consistent IDs for testing
3. The payment_hash would come from a real Lightning Network invoice
4. Real implementations should use persistent storage
5. Error handling is minimal in these examples

