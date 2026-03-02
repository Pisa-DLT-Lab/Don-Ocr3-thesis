#!/bin/sh

# Wait before running the nodes
echo " [WAIT] The hardhat deploy..."

sleep 20

# Message to confirm the start of a node
echo " [START] Tempo scaduto! Avvio il nodo Oracle..."

# Execute the original command passed by docker compose
exec "$@"