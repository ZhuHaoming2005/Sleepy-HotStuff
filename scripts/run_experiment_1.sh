#!/bin/bash

if [ "$#" -ne 2 ]; then
    echo -e "$0 <the specific storage option> <the waiting time (seconds) for the evaluation>\n"
    echo -e "    storage options:\n"
    echo -e "        1: storing all parameters\n"
    echo -e "        2: storing minimum parameters\n"
    echo -e "        3: storing no parameters\n"
    exit 1
fi

STORAGE_OPTION=$1
SLEEP_TIME=$2

if [ "$1" -eq 1 ]; then
  echo "Evaluating performance when storing all consensus parameters"
  EXPERIMENT_DIR="./experiments/HS_storage/all"
fi

if [ "$1" -eq 2 ]; then
  echo "Evaluating performance when storing minimum consensus parameters"
  EXPERIMENT_DIR="./experiments/HS_storage/minimum"
fi

if [ "$1" -eq 3 ]; then
  echo "Evaluating performance when storing none of the consensus parameters"
  EXPERIMENT_DIR="./experiments/HS_storage/none"
fi

mkdir -p "$EXPERIMENT_DIR"

# Remove the old log file
rm -rf "../var/log"

# modify the configuration file
echo
echo "[Configuration] modify the configuration file"
cp -v -f "${EXPERIMENT_DIR}/conf.json" "./etc/conf.json"
if [ $? -ne 0 ]; then
  echo "copying configuration file fails"
  exit 1
fi
sleep 1

# start 4 servers
echo
echo "[Start Server] start 4 servers"
for ((i=0; i <= 3; i++)); do
    echo "start replica: ./server $i"
    ./server $i &
done
sleep 1


# Start the client.
echo
echo "[Start Client] start the client."
./client 100 1 55000


# wait for the end of the evaluation
echo
echo "Evaluation in progress... waiting ${SLEEP_TIME} seconds."
sleep $SLEEP_TIME


# kill all servers and clients
echo
echo "[Kill Processes] kill all server and client"
killall server
killall client
# echo "The latencies of 55 blocks are expected to be output. If not, increase the SLEEP_TIME in the script $0"
sleep 1

echo
echo "[Output] Print the performance of the sleepy replica"
python3 "./scripts/getPerformanceData.py"

wait
