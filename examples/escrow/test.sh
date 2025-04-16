#!/bin/bash

echo "Full happy-path test for Catallax escrow system (NIP-3400)"
echo ""

## Generate Test Keys

# Generate keys for each party (save these for the test session)
echo "Generating arbiter key..."
ARBITER_KEY=$(nak key generate)
ARBITER_PUB=$(nak key public $ARBITER_KEY)
echo "Arbiter pubkey: $ARBITER_PUB"
echo ""

echo "Generating patron key..."
PATRON_KEY=$(nak key generate)
PATRON_PUB=$(nak key public $PATRON_KEY)
echo "Patron pubkey: $PATRON_PUB"
echo ""

echo "Generating worker key..."
WORKER_KEY=$(nak key generate)
WORKER_PUB=$(nak key public $WORKER_KEY)
echo "Worker pubkey: $WORKER_PUB"
echo ""

### 1. Register Arbiter Service (Kind 33400)

# Register the arbiter service announcement
echo "Register an arbiter service"
echo ""
SERVICE_ID="web-dev-escrow-service"
ARBITER_EVENT=$(nak event --sec $ARBITER_KEY --kind 33400 --content "{
  \"name\": \"Trusted Escrow Service\",
  \"about\": \"Professional escrow service for nostr tasks\",
  \"policy_text\": \"Mediation first, then arbitration if needed\"
}" -t "d=$SERVICE_ID" -p $ARBITER_PUB -t "fee_type=percentage" -t "fee_amount=0.05" -t "min_amount=10000" -t "t=programming" -t "t=web development" ws://localhost:3334)

echo $ARBITER_EVENT
echo ""

# Save the event ID
ARBITER_EVENT_ID=$(echo $ARBITER_EVENT | jq -r .id)
echo "Arbiter service announcement event ID: $ARBITER_EVENT_ID"
echo ""

### 2. Create Task Proposal (Kind 33401)

# Create a task proposal (status: proposed)
echo "Create task proposal"
echo ""
TASK_ID="landing-page-task-123"
DEADLINE=$(date -d "+7 days" +%s)
TASK_EVENT=$(nak event --sec $PATRON_KEY --kind 33401 --content "{
  \"title\": \"Build a simple landing page\",
  \"description\": \"Create a responsive landing page for a small business\",
  \"requirements\": \"HTML/CSS/JS, responsive design, contact form, 3 sections\",
  \"deadline\": $DEADLINE
}" -t "d=$TASK_ID" -p $PATRON_PUB -p $ARBITER_PUB -t "a=33400:$ARBITER_PUB:$SERVICE_ID" -t "amount=100000" -t "status=proposed" -t "t=web development" ws://localhost:3334)

echo $TASK_EVENT
echo ""

# Save the event ID
TASK_EVENT_ID=$(echo $TASK_EVENT | jq -r .id)
echo "Task proposal event ID: $TASK_EVENT_ID"
echo ""

### 3. Update Task to Funded Status (Kind 33401 replacement)

# Simulate a zap to fund the escrow
ZAP_RECEIPT_ID="zap_receipt_123" # In reality this would come from a real zap

# Update task to funded status
echo "Update task to funded status"
echo ""
FUNDED_EVENT=$(nak event --sec $PATRON_KEY --kind 33401 --content "{
  \"title\": \"Build a simple landing page\",
  \"description\": \"Create a responsive landing page for a small business\",
  \"requirements\": \"HTML/CSS/JS, responsive design, contact form, 3 sections\",
  \"deadline\": $DEADLINE
}" -t "d=$TASK_ID" -p $PATRON_PUB -p $ARBITER_PUB -t "a=33400:$ARBITER_PUB:$SERVICE_ID" -t "amount=100000" -t "status=funded" -t "t=web development" -t "e=$ZAP_RECEIPT_ID:relay-url:zap" ws://localhost:3334)

echo $FUNDED_EVENT
echo ""

# Save the event ID
FUNDED_EVENT_ID=$(echo $FUNDED_EVENT | jq -r .id)
echo "Funded task event ID: $FUNDED_EVENT_ID"
echo ""

### 4. Update Task to In Progress with Worker Assigned (Kind 33401 replacement)

# Update task to in_progress status with worker assigned
echo "Update task to in_progress status with worker assigned"
echo ""
PROGRESS_EVENT=$(nak event --sec $PATRON_KEY --kind 33401 --content "{
  \"title\": \"Build a simple landing page\",
  \"description\": \"Create a responsive landing page for a small business\",
  \"requirements\": \"HTML/CSS/JS, responsive design, contact form, 3 sections\",
  \"deadline\": $DEADLINE
}" -t "d=$TASK_ID" -p $PATRON_PUB -p $ARBITER_PUB -p $WORKER_PUB -t "a=33400:$ARBITER_PUB:$SERVICE_ID" -t "amount=100000" -t "status=in_progress" -t "t=web development" -t "e=$ZAP_RECEIPT_ID:relay-url:zap" ws://localhost:3334)

echo $PROGRESS_EVENT
echo ""

# Save the event ID
PROGRESS_EVENT_ID=$(echo $PROGRESS_EVENT | jq -r .id)
echo "In-progress task event ID: $PROGRESS_EVENT_ID"
echo ""

### 5. Update Task to Submitted Status (Kind 33401 replacement)

# Update task to submitted status
echo "Update task to submitted status"
echo ""
SUBMITTED_EVENT=$(nak event --sec $PATRON_KEY --kind 33401 --content "{
  \"title\": \"Build a simple landing page\",
  \"description\": \"Create a responsive landing page for a small business\",
  \"requirements\": \"HTML/CSS/JS, responsive design, contact form, 3 sections\",
  \"deadline\": $DEADLINE
}" -t "d=$TASK_ID" -p $PATRON_PUB -p $ARBITER_PUB -p $WORKER_PUB -t "a=33400:$ARBITER_PUB:$SERVICE_ID" -t "amount=100000" -t "status=submitted" -t "t=web development" -t "e=$ZAP_RECEIPT_ID:relay-url:zap" ws://localhost:3334)

echo $SUBMITTED_EVENT
echo ""

# Save the event ID
SUBMITTED_EVENT_ID=$(echo $SUBMITTED_EVENT | jq -r .id)
echo "Submitted task event ID: $SUBMITTED_EVENT_ID"
echo ""

### 6. Task Conclusion (Kind 3402)

# Simulate a zap for worker payment
PAYOUT_ZAP_ID="payout_zap_receipt_456" # In reality this would come from a real zap

# Arbiter concludes the task with successful resolution
echo "Arbiter concludes task"
echo ""
CONCLUDE_EVENT=$(nak event --sec $ARBITER_KEY --kind 3402 --content "{
  \"resolution_details\": \"Task completed successfully, landing page delivered with all requirements met.\"
}" -t "e=$PAYOUT_ZAP_ID" -t "e=$SUBMITTED_EVENT_ID" -p $PATRON_PUB -p $ARBITER_PUB -p $WORKER_PUB -t "resolution=successful" -t "a=33401:$PATRON_PUB:$TASK_ID" ws://localhost:3334)

echo $CONCLUDE_EVENT
echo ""

# Save the event ID
CONCLUDE_EVENT_ID=$(echo $CONCLUDE_EVENT | jq -r .id)
echo "Task conclusion event ID: $CONCLUDE_EVENT_ID"
echo ""

## Query Events

# Query all escrow-related events
echo "All arbiter service announcements:"
nak req -k 33400 ws://localhost:3334
echo ""

echo "All task proposals:"
nak req -k 33401 ws://localhost:3334
echo ""

echo "All task conclusions:"
nak req -k 3402 ws://localhost:3334
echo ""

# Query complete task thread
echo "Complete thread for task:"
nak req --id $TASK_EVENT_ID --id $FUNDED_EVENT_ID --id $PROGRESS_EVENT_ID --id $SUBMITTED_EVENT_ID --id $CONCLUDE_EVENT_ID ws://localhost:3334
echo ""

echo "Test script completed!"