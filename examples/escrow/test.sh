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
AGENT_EVENT=$(nak event --sec $AGENT_KEY --kind 3400 --content "{
  \"name\": \"Trusted Escrow Agent\",
  \"about\": \"Professional escrow service for nostr tasks\",
  \"fee_rate\": 0.01,
  \"min_amount\": 1000,
  \"max_amount\": 1000000,
  \"dispute_resolution_policy\": \"Mediation first, then arbitration\",
  \"supported_currencies\": [\"BTC\"]
}" -p $AGENT_PUB -t r="https://terms.example.com" ws://localhost:3334)

# Save the event ID
AGENT_EVENT_ID=$(echo $AGENT_EVENT | jq -r .id)
echo "Agent registration event ID: $AGENT_EVENT_ID"
echo ""

### 2. Create Task Proposal

# Create a task proposal
DEADLINE=$(date -d "+7 days" +%s)
TASK_EVENT=$(nak event --sec $CREATOR_KEY --kind 3401 --content "{
  \"description\": \"Create a nostr client\",
  \"requirements\": \"Must support NIPs 1,2,4\",
  \"deadline\": $DEADLINE
}" -p $CREATOR_PUB -p $AGENT_PUB -t amount=100000 ws://localhost:3334)

# Save the event ID
TASK_EVENT_ID=$(echo $TASK_EVENT | jq -r .id)
echo "Task proposal event ID: $TASK_EVENT_ID"
echo ""

### 3. Agent Accepts Task

# Agent accepts the task
ACCEPT_EVENT=$(nak event --sec $AGENT_KEY --kind 3402 -e $TASK_EVENT_ID -p $CREATOR_PUB -p $AGENT_PUB ws://localhost:3334)

# Save the event ID
ACCEPT_EVENT_ID=$(echo $ACCEPT_EVENT | jq -r .id)
echo "Task acceptance event ID: $ACCEPT_EVENT_ID"
echo ""

### 4. Task Finalization (after zap)

# Simulate task finalization after zap
ZAP_RECEIPT_ID="zap_receipt_123" # In reality this would come from a real zap
FINAL_EVENT=$(nak event --sec $CREATOR_KEY --kind 3403 -e $ACCEPT_EVENT_ID -e $ZAP_RECEIPT_ID -p $CREATOR_PUB -p $AGENT_PUB -t amount=100000 ws://localhost:3334)

# Save the event ID
FINAL_EVENT_ID=$(echo $FINAL_EVENT | jq -r .id)
echo "Task finalization event ID: $FINAL_EVENT_ID"
echo ""

### 5. Worker Application

# Worker applies for the task
APPLY_EVENT=$(nak event --sec $WORKER_KEY --kind 3404 --content "I would like to work on this task. I have experience building nostr clients." -e $FINAL_EVENT_ID -p $CREATOR_PUB -p $AGENT_PUB ws://localhost:3334)

# Save the event ID
APPLY_EVENT_ID=$(echo $APPLY_EVENT | jq -r .id)
echo "Worker application event ID: $APPLY_EVENT_ID"
echo ""

### 6. Worker Assignment

# Creator assigns the task to worker
ASSIGN_EVENT=$(nak event --sec $CREATOR_KEY --kind 3405 -e $FINAL_EVENT_ID -e $APPLY_EVENT_ID -p $WORKER_PUB -p $AGENT_PUB ws://localhost:3334)

# Save the event ID
ASSIGN_EVENT_ID=$(echo $ASSIGN_EVENT | jq -r .id)
echo "Worker assignment event ID: $ASSIGN_EVENT_ID"
echo ""

### 7. Work Submission

# Worker submits completed work
SUBMIT_EVENT=$(nak event --sec $WORKER_KEY --kind 3406 --content "Work completed. Repository: https://github.com/example/nostr-client" -e $ASSIGN_EVENT_ID -p $CREATOR_PUB -p $AGENT_PUB ws://localhost:3334)

# Save the event ID
SUBMIT_EVENT_ID=$(echo $SUBMIT_EVENT | jq -r .id)
echo "Work submission event ID: $SUBMIT_EVENT_ID"
echo ""

### 8. Task Resolution

# Agent resolves the task after verifying work and processing payment
RESOLVE_EVENT=$(nak event --sec $AGENT_KEY --kind 3407 --content "{
  \"resolution\": \"completed\",
  \"resolution_details\": \"Work verified and payment sent to worker\"
}" -e $SUBMIT_EVENT_ID -e $ZAP_RECEIPT_ID -p $CREATOR_PUB -p $WORKER_PUB -t "amount=99000" ws://localhost:3334)

# Save the event ID
RESOLVE_EVENT_ID=$(echo $RESOLVE_EVENT | jq -r .id)
echo "Task resolution event ID: $RESOLVE_EVENT_ID"
echo ""

## Query Events

# Query all escrow-related events
echo "All escrow agent registrations:"
nak req -k 3400 ws://localhost:3334
echo ""

echo "All task proposals:"
nak req -k 3401 ws://localhost:3334
echo ""

echo "All agent acceptances:"
nak req -k 3402 ws://localhost:3334
echo ""

echo "All task finalizations:"
nak req -k 3403 ws://localhost:3334
echo ""

echo "All worker applications:"
nak req -k 3404 ws://localhost:3334
echo ""

echo "All worker assignments:"
nak req -k 3405 ws://localhost:3334
echo ""

echo "All work submissions:"
nak req -k 3406 ws://localhost:3334
echo ""

echo "All task resolutions:"
nak req -k 3407 ws://localhost:3334
echo ""

# Query complete task thread
echo "Complete thread for task:"
nak req --id $TASK_EVENT_ID --id $ACCEPT_EVENT_ID --id $FINAL_EVENT_ID --id $APPLY_EVENT_ID --id $ASSIGN_EVENT_ID --id $SUBMIT_EVENT_ID --id $RESOLVE_EVENT_ID ws://localhost:3334
echo ""

echo "Test script completed!"
