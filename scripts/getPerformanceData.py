
import time
import re
import traceback

# In: Eva_log

# Format: 2024/03/27 13:26:47 [Replica] Processed %d (ops=%d, clockTime=%d ms, seq=%v) operations using %d ms. Throughput %d tx/s.
# example_log_string = "2024/00/00 00:00:00 [Replica] Processed 100 (ops=200, clockTime=100 ms, seq=300) operations using 400 ms. Throughput 500 tx/s."
def processHotstuffData(n, option, ss, date=''):
    #print("n:", n)
    commitLatency = []
    throughput = []
    clockTime = {}
    for ID in range(0, n):
        if date == '':
            filename = "../var/log/" + str(ID) + "/" + time.strftime("%Y%m%d", time.localtime()) + "_Eva.log"
        else:
            filename = "../var/log/" + str(ID) + "/" + date + "_Eva.log"
        try:
            with open(filename, 'r') as file:
                latency = []
                clockTime[ID] = []
                if option == 2:
                    throughput = []
                while True:
                    line = file.readline()
                    #print(line)
                    if not line:
                        break
                    numbers = re.findall(r'\d+', line)
                    t = int(numbers[-1])
                    l = int(numbers[-2])
                    seq = int(numbers[-3])
                    c = int(numbers[-4])
                    #print(c, seq, l, t)
                    # print("Numbers:", numbers)

                    latency.append(l)
                    if seq > 2:
                        # if l < 100:
                        #     continue
                        throughput.append(t)
                        clockTime[ID].append(c)
                    if option == 1 and seq >= 5:
                        # Note: the log data must be ensured that the `seq` increases one by one.
                        cl = latency[seq - 1] + latency[seq - 2] + latency[seq - 3]
                        commitLatency.append(cl)
                file.close()

                if option == 2:
                    pass
                    # filename = "fetchedLogs/" + str(ID) + "/" + time.strftime("%Y%m%d",
                    #                                                           time.localtime()) + "_s" + str(
                    #     ss) + "_rawThroughputTime.csv"
                    # with open(filename, 'w', newline='') as rawfile:
                    #     writer = csv.writer(rawfile)
                    #     output = zip(clockTime[ID], throughput)
                    #     writer.writerows(output)
        except ValueError as e:
            print("exception:", e)
            traceback.print_exc()
        except Exception as e:
            print("exception:", e)
            traceback.print_exc()

    if option == 1:
        #print(commitLatency)
        #print(throughput)
        return sum(throughput) / len(throughput), sum(commitLatency) / len(commitLatency)
    elif option == 2:
        print("The data has been processed.")
        # print(clockTime,throughput)
        # outputTimelineData(id, clockTime[id], throughput)
        # outputTimelineData(clockTime[n-1], throughput)
    else:
        print("Error: unsupported option!")



if __name__ == '__main__':
    
    tps, latency = processHotstuffData(4, 1, 0)
    print(f"throughput(tps):{tps}, latency(ms):{latency}")
