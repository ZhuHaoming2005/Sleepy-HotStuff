#!/bin/bash

EXPERIMENT_DIR="./experiments/double_spending/HS-nSS"
mkdir -p "$EXPERIMENT_DIR"
OUTPUT_DIR="${EXPERIMENT_DIR}/output"
mkdir -p "$OUTPUT_DIR"

# modify the configuration file
echo
echo "[Configuration] modify the configuration file"
cp -v -f "${EXPERIMENT_DIR}/conf.json" "./etc/conf.json"
if [ $? -ne 0 ]; then
  echo "copying configuration file fails"
  exit 1
fi

sleep 1


echo
echo "[Start Server] start 4 servers and redirect their outputs to output/"
# start 4 servers and redirect their outputs to output/
for ((i=0; i <= 3; i++)); do
    LOG_FILE="${OUTPUT_DIR}/server_${i}.log"
    echo "start replica: ./server $i, the output is in $LOG_FILE"
    ./server $i > "$LOG_FILE" 2>&1 &
done
sleep 1


# start the client and send two double-spending transactions.
echo
echo "[Start Client] start the client and send two double-spending transactions."
./client 100 0 1 f0t1v40f1t2v40
sleep 5
echo
./client 100 0 1 f0t2v40

# wait for the end of the attack
sleep 5

echo
echo "[Output] Print the output of the sleepy replica"
cat "${OUTPUT_DIR}/server_2.log"
sleep 1

# kill all server and client
echo
echo "[Kill Process] kill all server and client"
killall server
killall client

wait