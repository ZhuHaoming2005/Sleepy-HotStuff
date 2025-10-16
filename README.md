# Sleepy-HotStuff

## 系统要求

需要具有以下最低规格的机器：

- 操作系统: Ubuntu Server 22.04 LTS (amd64)
- CPU: 4 核心
- 内存: 16 GB
- 磁盘: 40 GB 可用空间 (推荐SSD)

## 先决条件

在开始之前，您需要更新系统的包列表并安装⼀些基本⼯具，例如`git`，`wget`和`python3-pip`。

打开终端并运⾏以下命令：

```bash
# 更新包列表
sudo apt update -y

# 安装基本的构建工具 
sudo apt install -y python3-pip git wget zip unzip vim curl psmisc
```

## 安装

安装过程主要分为两个步骤：安装 Go 编程语⾔、构建项⽬。

### 第 1 步：安装 Go （GoLang）

此项⽬需要 Go 版本`1.19.4`或更⾼版本。

1.  **下载 Go ⼆进制存档**
    您可以从 Go 官⽅⽹站下载。如果您位于访问速度较慢的地区，则可以使用镜像。

    *   **官⽅链接：**
        ```bash
        wget https://go.dev/dl/go1.19.4.linux-amd64.tar.gz
        ```
    *   **阿⾥云镜像：**
        ```bash
        wget https://mirrors.aliyun.com/golang/go1.19.4.linux-amd64.tar.gz
        ```

2.  **解压存档**
    此命令会将 Go 安装到您选择的⽬录中。

     ```bash
    # 将 [your directory] 替换为您选择的目录。
    tar -C [your directory] -xzf go1.19.4.linux-amd64.tar.gz
    ```

    两个例⼦：
    ```bash
    # 示例 1：系统范围的安装（需要 sudo）
    sudo tar -C /usr/local -xzf go1.19.4.linux-amd64.tar.gz
    
    # 示例 2：主目录中的用户级安装 (~)
    tar -C ~ -xzf go1.19.4.linux-amd64.tar.gz
    ```

3.  **将 Go 添加到您的 PATH。**
    这是**关键步骤**，以便您可以在终端使用 `go` 命令。

    ```bash
    # 将 [your directory] 替换为您下载的位置。
    echo 'export PATH=[your directory]/go/bin:$PATH' >> ~/.bashrc
    ```
    两个例⼦：
    ```bash
    # 如果你将 Go 安装在 /usr/local：
    echo 'export PATH=/usr/local/go/bin:$PATH' >> ~/.bashrc

    # 如果你将 Go 安装在主目录（~）：
    echo 'export PATH=~/go/bin:$PATH' >> ~/.bashrc
    ```
    载入更新后的 .bashrc 配置：
    ```bash
    source ~/.bashrc
    ```
    
    为验证安装是否成功，输入：
    ```bash
    go version
    # 应输出：go version go1.19.4 linux/amd64
    ```

### 第 2 步：克隆并构建项目

现在已安装 Go，您可以克隆仓库并编译源代码。
1.  **进入项目目录。**
    ```bash
    cd Sleepy-HotStuff
    ```

2.  **切换到稳定版本。**
    ```bash
    git checkout ndss-ae-2
    ```

3.  **运行构建脚本。**
    该仓库提供两种构建脚本，可按需选择。

    *   **选项 A：在线构建**
        该脚本会在编译前自动从网络下载所有所需依赖。
        ```bash
        ./scripts/build.sh
        ```

    *   **选项 B：离线构建**
        该脚本使用仓库内 `vendor/` 目录中已包含的依赖。它更快且无需联网获取依赖，适合可复现实验或受网络限制的环境。
        ```bash
        ./scripts/build_offline.sh
        ```

我们**推荐**使用离线构建以确保顺畅且可靠的编译过程。

构建成功后，您将在项目根目录下看到可执行文件（`server`、`client`、`ecdsagen`）。

## 使用

构建成功后，您可以在项目根目录运行编译好的二进制文件。

### 运行服务器

可以在 `etc/conf.json` 中配置服务器实例。启动服务器前，需在该文件中设置其 ID、主机与端口。

1.  **配置服务器副本**

    打开 `etc/conf.json`，编辑 `replicas` 数组。数组中的每个对象代表一个可启动的服务器实例。

    ```json
    "replicas": [
        {
          "id": "0",            // 该服务器实例的唯一 ID
          "host": "127.0.0.1",  // 服务器监听的 IP 地址
          "port": "11000"       // 端口号
        },
        ...
    ]
    ```

2.  **启动服务器实例**

    使用以下命令启动服务器，其中 `[id]` 必须与 `etc/conf.json` 中配置的某个 ID 匹配。

    ```bash
    ./server [id]
    ```

    **示例：**
    该命令启动 `id` 为 0 的服务器实例。
    ```bash
    ./server 0
    ```

### 运行客户端

客户端可用于向正在运行的服务器发送请求。

1.  **命令语法**

    按以下格式运行客户端：

    ```bash
    ./client [client-id] [operation-type] [batch-size]
    ```

2.  **参数**

    | 参数               | 类型/取值              | 说明                                                                                                      |
    | ------------------ | ---------------------- | --------------------------------------------------------------------------------------------------------- |
    | `[client-id]`      | Integer                | 用于标识该客户端的唯一正整数。不得与任何服务器副本 ID 冲突。                                              |
    | `[operation-type]` | `0` 或 `1`             | 指定要执行的操作类型：<br> • `0`：单次写入。<br> • `1`：批量写入。                                         |
    | `[batch-size]`     | Integer                | 要执行的操作数量。单次写入通常为 `1`；批量写入为批量大小。                                                |

3.  **示例**

    *   **执行单次写入：**
        该命令启动 ID 为 `100` 的客户端并发送一次写请求。
        ```bash
        ./client 100 0 1
        ```

    *   **执行批量写入：**
        该命令启动 ID 为 `200` 的客户端并发送 500 条写请求。
        ```bash
        ./client 200 1 500
        ```

### 一个可运行的示例：启动四个服务器与一个客户端

您可以直接使用提供的 `etc/conf.json`（已预配置为四个服务器），或按需修改，但需确保 `replicas` 数组仍定义四个副本。

例如：

```json
"replicas": [
    {
        "id": "0",          
        "host": "127.0.0.1",
        "port": "11000"
    },
    {
        "id": "1",
        "host": "127.0.0.1",
        "port": "11001"
    },
    {
        "id": "2",
        "host": "127.0.0.1",
        "port": "11002"
    },
    {
        "id": "3",
        "host": "127.0.0.1",
        "port": "11003"
    }
]
```

然后在四个终端中分别启动四个服务器：

```bash
    # 1号终端:
    ./server 0

    # 2号终端:
    ./server 1

    # 3号终端:
    ./server 2

    # 4号终端:
    ./server 3
```

每个服务器的预期输出如下：

```
13:24:58 **Starting replica 0
13:24:58 the local database has started
homepath %s /home/ubuntu/Sleepy-HotStuff
homepath %s /home/ubuntu/Sleepy-HotStuff
13:24:58 Use ECDSA for authentication
13:24:58 sleeptimer value 50
13:24:58 running HotStuff
13:24:58 Starting sender 0
homepath %s /home/ubuntu/Sleepy-HotStuff
13:24:58 starting connection manager
13:24:58 ready to listen to port :11000
```

当所有服务器都监听各自的端口后，启动一个客户端：

```bash
./client 100 0 1
```

客户端的预期输出如下：

```
2025/08/26 13:31:21 Rtype 0
2025/08/26 13:31:21 Starting client test
2025/08/26 13:31:21 ** Client 100
homepath %s /home/ubuntu/Sleepy-HotStuff
2025/08/26 13:31:21 Client 100 started.
homepath %s /home/ubuntu/Sleepy-HotStuff
2025/08/26 13:31:21 Use ECDSA for authentication
2025/08/26 13:31:21 starting connection manager
2025/08/26 13:31:21 len of request:  54
2025/08/26 13:31:21 Got a reply rep
2025/08/26 13:31:21 Got a reply rep
2025/08/26 13:31:21 Got a reply rep
2025/08/26 13:31:21 Got a reply rep
2025/08/26 13:31:21 Done with all client requests.
```

如果直接使用提供的 `etc/conf.json`，每个服务器的预期输出如下：

```
13:31:22 [!!!] Ready to output a value for height 1
13:31:22 [!!!] Ready to output a value for height 2
13:31:22 [!!!] Ready to output a value for height 3
13:31:22 [!!!] Ready to output a value for height 4
13:31:22 [!!!] Ready to output a value for height 5
13:31:22 [!!!] Ready to output a value for height 6
13:31:22 [!!!] Ready to output a value for height 7
13:31:22 [!!!] Ready to output a value for height 8
13:31:22 [!!!] Ready to output a value for height 9
13:31:22 [!!!] Ready to output a value for height 10
```

如果该服务器是领导者（leader），预期输出如下：
```
13:31:21 batchSize: 1
13:31:21 proposing block with height 1, awaiting 1 blocks
13:31:22 batchSize: 0
13:31:22 proposing block with height 2, awaiting 2 blocks
13:31:22 batchSize: 0
13:31:22 proposing block with height 3, awaiting 3 blocks
13:31:22 batchSize: 0
13:31:22 proposing block with height 4, awaiting 4 blocks
13:31:22 batchSize: 0
13:31:22 proposing block with height 5, awaiting 5 blocks
13:31:22 batchSize: 0
13:31:22 proposing block with height 6, awaiting 6 blocks
13:31:22 batchSize: 0
13:31:22 proposing block with height 7, awaiting 7 blocks
13:31:22 batchSize: 0
13:31:22 proposing block with height 8, awaiting 8 blocks
13:31:22 batchSize: 0
13:31:22 proposing block with height 9, awaiting 9 blocks
13:31:22 batchSize: 0
13:31:22 proposing block with height 10, awaiting 10 blocks
```

最后，通过以下命令终止所有服务器与客户端：

```bash
killall server
killall client
```

## 评估

我们提供自动化演示脚本。所有命令均应在项目根目录执行。我们提供小规模的单机实验，用于完成评估、演示。

### 实验 1：不同存储选项下的性能

该实验评估 HotStuff 在不同存储选项下的性能。

本实验在本机运行 4 个服务器副本与 1 个客户端进程。每个副本的详细日志存放于 `var` 目录，该目录与项目根目录（`Sleepy-HotStuff`）同级。具体地，副本 N 的输出日志位于 `var/log/N/date_Eva.log`。

#### 实验 1.1：所有参数存于稳定存储

该子实验测量在将所有共识参数存入稳定存储时 HotStuff 的延迟与吞吐。

**操作步骤**

在项目根目录运行以下脚本：

```bash
./scripts/run_experiment_1.sh 1 50
```

*   第一个参数 `1` 选择该子实验（所有参数存于稳定存储）。
*   第二个参数 `50` 指定脚本等待实验完成的时长（秒）。

**预期结果**

脚本会先显示配置切换、启动 4 个服务器进程、再启动客户端。脚本运行期间将看到类似如下的输出：

```
Evaluating performance when storing all consensus parameters

[Configuration] modify the configuration file
'./experiments/HS_storage/all/conf.json' -> './etc/conf.json'

[Start Server] start 4 servers
...
[Start Client] start the client.
...
Evaluation in progress... waiting 50 seconds.
...
[Replica] Processed 55000 (ops=1000, clockTime=37448 ms, seq=55) operations using 1230 ms. Throughput 813 tx/s. 
...
[Kill Processes] kill all server and client
```

在等待指定时间后，脚本会终止所有进程并计算平均性能指标。关键在于**最后一行**，它给出整体吞吐与延迟：

```
[Output] Print the performance of the sleepy replica
throughput(tps):2027.004716981132, latency(ms):2114.544117647059
```

> **注意：** 吞吐与延迟的具体数值可能因硬件与负载略有差异。若未打印 `seq=55` 的性能行，可能是等待时间不足，请尝试使用更长的等待时间（如 `./scripts/run_experiment_1.sh 1 70`）。

#### 实验 1.2：最少参数存于稳定存储

该子实验评估仅将最少共识参数存入稳定存储时的性能。

**操作步骤**

在项目根目录运行以下命令：

```bash
./scripts/run_experiment_1.sh 2 30
```

*   第一个参数 `2` 选择该子实验（最少参数存于稳定存储）。
*   第二个参数 `30` 为等待时间（秒）。

**预期结果**

```
Evaluating performance when storing minimum consensus parameters

[Configuration] modify the configuration file
'./experiments/HS_storage/minimum/conf.json' -> './etc/conf.json'

[Start Server] start 4 servers
...
[Start Client] start the client.
...
Evaluation in progress... waiting 30 seconds.
...
13:11:27 [Replica] Processed 2000 (ops=1000, clockTime=54 ms, seq=2) operations using 54 ms. Throughput 18518 tx/s. 
...
13:11:27 [!!!] Ready to output a value for height 10
...
13:11:37 [Replica] Processed 55000 (ops=1000, clockTime=10284 ms, seq=55) operations using 409 ms. Throughput 2444 tx/s.
...
[Kill Processes] kill all server and client
```

实验结束后，脚本会计算并打印平均性能结果。重点关注**最后一行**：

```
[Output] Print the performance of the sleepy replica
throughput(tps):8046.622641509434, latency(ms):575.2696078431372
```

#### 实验 1.3：无参数存于稳定存储

该子实验评估在不将任何共识参数存入稳定存储时的性能。

**操作步骤**

在项目根目录运行以下命令：

```bash
./scripts/run_experiment_1.sh 3 30
```

*   第一个参数 `3` 选择该子实验（不存储任何参数）。
*   第二个参数 `30` 为等待时间（秒）。

**预期结果**

```
Evaluating performance when storing none of the consensus parameters

[Configuration] modify the configuration file
'./experiments/HS_storage/none/conf.json' -> './etc/conf.json'

[Start Server] start 4 servers
...
[Start Client] start the client.
...
Evaluation in progress... waiting 30 seconds.
...
13:11:37 [Replica] Processed 55000 (ops=1000, clockTime=10284 ms, seq=55) operations using 56 ms. Throughput 17857 tx/s.
...
[Kill Processes] kill all server and client
```

实验结束后，脚本会计算并打印平均性能结果。重点关注**最后一行**：

```
[Output] Print the performance of the sleepy replica
throughput(tps):17784.372641509435, latency(ms):168.97549019607843
```

### 实验 2：双花攻击

本实验将演示一个去中心化支付系统，用于测试我们协议抵御双花攻击的能力。

#### 实验 2.1：攻击无稳定存储的 HotStuff

该子实验表明不在稳定存储中保存共识参数的标准 HotStuff 实现易受双花攻击。

**操作步骤**

在项目根目录运行以下脚本：

```bash
./scripts/run_experiment_2_1.sh
```

该脚本将：
1.  将系统配置为无稳定存储的 HotStuff。
2.  启动 4 个副本。每个副本的输出会重定向到独立的日志文件（如 `experiments/double_spending/HS-nSS/output/server_0.log`）。
3.  模拟客户端发送两笔相互冲突的交易（账户 `0` -> `1` 与账户 `0` -> `2`）。
4.  使其中一个副本（副本 2）“睡眠”并“唤醒”，以模拟重启。

**预期结果**

脚本首先会显示配置与客户端提交交易的过程。关键部分是瞌睡副本（副本 2）的日志，日志会打印到控制台。

您应观察到如下事件序列，以确认**双花攻击成功**：

1.  **首先，副本在进入睡眠前，在高度为 1 的区块中提交第一笔交易（`0 -> 1`）。**
    注意包含交易 `{"From":"0","To":"1","Value":40}` 的区块输出，随后会出现 “Falling asleep” 提示。

    ```
    13:39:41 [!!!] Ready to output a value for height 2
    ...
    13:39:41 {"View":0,"Height":1,"TXS":[...,{"ID":100,"TX":{"From":"0","To":"1","Value":40"},,"Timestamp":1752586781315}]}
    ...
    13:39:41 Falling asleep in sequence 5...
    ```

2.  **重启后，该副本遗忘之前的状态，并在高度为 1 的区块中提交一笔冲突交易（`0 -> 2`）。**
    注意 “Wake up” 提示，随后会有一个新的已提交区块，其中包含交易 `{"From":"0","To":"2","Value":40}`。

    ```
    13:39:41 sleepTime: 3000 ms
    13:39:44 Wake up...
    13:39:44 Start the recovery process.
    13:39:44 recover to READY
    13:39:46 [!!!] Ready to output a value for height 1
    ...
    13:39:46 {"View":0,"Height":1,"TXS":[...,{"ID":100,"TX":{"From":"0","To":"2","Value":40"},"Timestamp":1752586786370}]}
    ```

该序列表明瞌睡副本在相同高度上提交了两个冲突区块，证实了无稳定存储的 HotStuff 存在双花漏洞。

#### 实验 2.2：攻击 Sleepy HotStuff-MinSS

该子实验展示我们的 Sleepy HotStuff-MinSS 协议可以抵御双花攻击。

**操作步骤**

在项目根目录运行以下脚本：

```bash
./scripts/run_experiment_2_2.sh
```

该脚本将：
1.  将系统配置为 Sleepy HotStuff-MinSS。
2.  启动 4 个副本。每个副本的输出会重定向到独立的日志文件（如 `experiments/double_spending/HotStuff-MinSS/output/server_0.log`）。
3.  模拟客户端发送两笔相互冲突的交易（账户 `0` -> `1` 与账户 `0` -> `2`）。
4.  使其中一个副本（副本 2）“睡眠”并“唤醒”，以模拟重启。

**预期结果**

1.  **入睡前：** 副本提交第一笔交易（`0 -> 1`）。
    ```
    13:58:35 [!!!] Ready to output a value for height 2
    ...
    13:58:35 {"View":0,"Height":1,"TXS":[...,{"ID":100,"TX":{"From":"0","To":"1","Value":40"},"Timestamp":1752587915411}]}
    ...
    13:58:35 Falling asleep in sequence 5...
    ```

2.  **重启后：** 副本依据稳定存储中的参数安全恢复。它知道自己曾处于视图 0，并发起到视图 1 的视图变更。
    ```
    13:58:38 Wake up...
    13:58:38 Start the recovery process.
    13:58:38 recover to the view 1
    13:58:38 Starting view change to view 1
    ```

3.  **攻击失败：** 恢复后的日志显示，视图 0 的区块（包括包含第一笔交易 0 -> 1 的区块）被保留在已提交账本中。
    
    ```
    13:58:45 [!!!] Ready to output a value for height 3
    13:58:45 {"View":0,"Height":0,"TXS":[{"ID":0,"TX":{"From":"","To":"0","Value":50},"Timestamp":0}]}
    13:58:45 {"View":0,"Height":1,"TXS":[...,{"ID":100,"TX":{"From":"0","To":"1","Value":40},"Timestamp":1752587915411}]}
    ...
    ```

#### 实验 2.3：攻击 Sleepy HotStuff-InMem

该子展示我们的 Sleepy HotStuff-InMem 协议可以抵御双花攻击。

**操作步骤**

在项目根目录运行以下脚本：

```bash
./scripts/run_experiment_2_3.sh
```

该脚本将：
1.  将系统配置为 Sleepy HotStuff-InMem。
2.  启动 6 个副本。每个副本的输出会重定向到独立的日志文件（如 `experiments/double_spending/HotStuff-InMem/output/server_0.log`）。
3.  模拟客户端发送两笔相互冲突的交易（账户 `0` -> `1` 与账户 `0` -> `2`）。
4.  使其中一个副本（副本 3）“睡眠”并“唤醒”，以模拟重启。

**预期结果**

1.  **入睡前：** 副本提交第一笔交易（`0 -> 1`）。
    ```
    14:17:50 [!!!] Ready to output a value for height 2
    14:17:50 {"View":0,"Height":0,"TXS":[{"ID":0,"TX":{"From":"","To":"0","Value":50},"Timestamp":0}]}
    14:17:50 {"View":0,"Height":1,"TXS":[...,{"ID":100,"TX":  {"From":"0","To":"1","Value":40},"Timestamp":1752589070118}]}
    ...
    14:17:50 Falling asleep in sequence 5...
    ```

2.  **重启后：** 副本被唤醒并启动 Sleepy HotStuff-InMem 恢复协议。您将看到与该协议相关的特定日志，例如 ECHO1 与 ECHO2 消息。
    ```
    14:17:54 Wake up...
    14:17:54 Start the recovery process.
    14:17:54 receive a ECHO1 msg from replica 0
    ...
    14:18:00 receive a TQC msg from replica 1 for view 0
    14:18:00 Starting view change to view 1
    ...
    14:18:10 receive a TQC msg from replica 5 for view 1
    14:18:10 Starting view change to view 2
    ...
    14:18:10 receive a ECHO2 msg from replica 4
    ...
    14:18:10 recover to READY
    ...
    ```

3.  **攻击失败：** 恢复后的日志显示，视图 0 的区块（包括包含第一笔交易 0 -> 1 的区块）被保留在已提交账本中。
    
    ```
    14:18:10 [!!!] Ready to output a value for height 198
    14:18:10 {"View":0,"Height":0,"TXS":[{"ID":0,"TX":{"From":"","To":"0","Value":50},"Timestamp":0}]}
    14:18:10 {"View":0,"Height":1,"TXS":[...,{"ID":100,"TX":{"From":"0","To":"1","Value":40},"Timestamp":1752589070118}]}
    ...
    ```
