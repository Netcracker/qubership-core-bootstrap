#!/bin/bash
LIMIT_PER_MIN=30
REQUESTS_PER_USER=50
USERS=2
DURATION_SEC=60

LIMIT_PER_SEC=$(echo "scale=2; $LIMIT_PER_MIN / 60" | bc)
MAX_ALLOWED=$(echo "$LIMIT_PER_SEC * $DURATION_SEC * $USERS" | bc)
TOTAL_REQUESTS=$((REQUESTS_PER_USER * USERS))

if (( $(echo "$TOTAL_REQUESTS > $MAX_ALLOWED" | bc -l) )); then
    REJECTED=$(echo "$TOTAL_REQUESTS - $MAX_ALLOWED" | bc)
    RATE=$(echo "scale=1; $REJECTED / $TOTAL_REQUESTS * 100" | bc)
    echo "Expected rejection rate: ${RATE}%"
else
    echo "Expected rejection rate: 0%"
fi