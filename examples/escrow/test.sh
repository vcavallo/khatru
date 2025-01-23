#!/bin/bash

echo "Full happy-path test for escrow NIP"
echo ""

## Generate Test Keys

# Generate keys for each party (save these for the test session)
echo "Generating escrow agent key..."
AGENT_KEY=$(nak key generate)
AGENT_PUB=$(nak key public $AGENT_KEY)
echo "Agent pubkey: $AGENT_PUB"
echo ""

echo "Generating task creator key..."
CREATOR_KEY=$(nak key generate)
CREATOR_PUB=$(nak key public $CREATOR_KEY)
echo "Creator pubkey: $CREATOR_PUB"
echo ""

echo "Generating worker key..."
WORKER_KEY=$(nak key generate)
WORKER_PUB=$(nak key public $WORKER_KEY)
echo "Worker pubkey: $WORKER_PUB"
echo ""

### 1. Register Escrow Agent

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
echo ""

### 2. Create Task

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
echo ""

### 3. Accept Task

# Worker accepts the task
ACCEPT_EVENT=$(nak event --sec $WORKER_KEY -k 30402 -c "{
  \"version\": \"1.0.0\",
  \"task_id\": \"$TASK_EVENT_ID\",
  \"worker_commitment\": \"I agree to complete this task according to requirements\"
}" -t e=$TASK_EVENT_ID -t p=$CREATOR_PUB -t p=$WORKER_PUB -t p=$AGENT_PUB ws://localhost:3334)

# Save the event ID
ACCEPT_EVENT_ID=$(echo $ACCEPT_EVENT | jq -r .id)
echo "Task acceptance event ID: $ACCEPT_EVENT_ID"
echo ""

### 4. Resolve Task

# Resolve the task (can be done by agent)
RESOLVE_EVENT=$(nak event --sec $AGENT_KEY -k 30403 -c "{
  \"version\": \"1.0.0\",
  \"task_id\": \"$TASK_EVENT_ID\",
  \"resolution\": \"settled\",
  \"settlement_proof\": \"lightning_txid_123\",
  \"resolution_details\": \"Task completed successfully\"
}" -t e=$TASK_EVENT_ID -t e=$ACCEPT_EVENT_ID -t p=$CREATOR_PUB -t p=$WORKER_PUB -t p=$AGENT_PUB ws://localhost:3334)

# Save the event ID
RESOLVE_EVENT_ID=$(echo $RESOLVE_EVENT | jq -r .id)
echo "Task resolution event ID: $RESOLVE_EVENT_ID"
echo ""

## Query Events

# Query all escrow-related events
echo "All escrow agent registrations:"
nak req -k 30400 ws://localhost:3334
echo ""

echo "All tasks:"
nak req -k 30401 ws://localhost:3334
echo ""

echo "All task acceptances:"
nak req -k 30402 ws://localhost:3334
echo ""

echo "All task resolutions:"
nak req -k 30403 ws://localhost:3334
echo ""

# Query specific task thread
echo "Complete thread for task $TASK_EVENT_ID:"
nak req --id $TASK_EVENT_ID --id $ACCEPT_EVENT_ID --id $RESOLVE_EVENT_ID localhost:3334
echo ""
echo "Examples done!"
