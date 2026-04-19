#!/bin/bash

SCRIPT_NAME="test/SimulateTransmit.js"
REPORT_SIZES=(100 200 500 1000)
for REPORT_SIZE in "${REPORT_SIZES[@]}"; do
    echo "Running test with report size: $REPORT_SIZE..."
    TMP_FILE="results_${REPORT_SIZE}.txt"
    node ${SCRIPT_NAME} --numOracles 16 --numFaulty 5 --numHolders 1000 --reportSize ${REPORT_SIZE} > ${TMP_FILE}
    RESULT=$(grep -e "Transmitted report for job ID: 0 with gas used:" ${TMP_FILE})
    echo $RESULT
done