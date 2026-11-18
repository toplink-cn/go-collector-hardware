# 主机资源采集器 v2

保持既有 Linux `collector` / Windows `collector.exe` 与 `HOST` 上报地址。每次计划任务执行一次，默认总预算 50 秒，采集预算 25 秒，每个工具命令最多 10 秒，HTTP 请求最多 5 秒；单次最多发送当前报告及两条历史报告。Go 1.20 或以上。

## 采样

- CPU 使用率取相邻 1 秒的计数差，按逻辑 CPU 编号保存，避免把超线程合并。Guest 不重复计入总时间，Idle/Iowait 视为空闲；拓扑变化和计数器重置保留失败证据。
- 内存为 bytes；`Memory.Free` 表示操作系统的 Available，避免 Linux 页缓存被计为不可用内存，也修复旧 Windows kB/bytes 混用。
- 网卡上报每接口累计 `Receive` / `Transmit` bytes；平台依真实采样时间求 bps，不在主机端伪造速率。
- CPU 温度来自实际传感器，ID 为稳定传感器标识、Label 为其名称，不假定与逻辑核一一对应；不支持时返回 unsupported。
- 磁盘最多 4 路并发；命令失败、设备缺失和 SMART 明确故障分别处理。SMART、容量、温度和通电时间缺失用 null。无序列号时保留真实设备 ID（包含 RAID type），不伪造序列号。
- IPMI/温度等可选能力缺失不会终止 CPU、内存、网卡采样。各模块有独立质量状态，顶层回收 panic，超时后保留已完成的指标。

`collection.version=2` 附 `sample_id`、整包 `collected_at`、各组 quality/state/reason/collected_at 和 units。网络等快指标保留自己的采样时间，不被慢速 SMART 结束时间覆盖。

## 上报与恢复

报告先以 0600 权限、fsync 和原子改名写入本地 pending，再发 HTTP。发送失败、超时、ACK 丢失或 ACK 身份不匹配时保留原 JSON 和 sample_id，下一次任务继续发送。

新版接收器只有事务提交后才能返回：

```json
{"success":true,"persisted":true,"sample_id":"与报告一致的ID"}
```

旧接收器的纯文本 `OK` 仅兼容本次实时报告，不能作为历史回放的可靠确认。队列满时先使用 OPTIONS 确认 `X-Collector-Protocol: 2`，再发送已有报告释放空间；不主动丢弃未确认数据。切换 HOST 后使用独立端点目录，避免旧数据发往另一平台。

默认队列最多 1440 条、64 MiB；达到上限会显式失败并保留已有数据。目录默认是可执行文件旁 `pending/<端点指纹>`；重装或升级必须保留 pending。Linux 用 flock、Windows 用 LockFileEx 防止本机同端点并发运行，异常退出由内核释放锁。

| 环境变量 | 默认值 |
| --- | --- |
| `HOST` | 必填 HTTP(S) 接收地址 |
| `COLLECTOR_SPOOL_DIR` | 可执行文件旁的 pending 根目录 |
| `COLLECTOR_SAMPLE_TIMEOUT` | 25s，可设 1s–30s |
| `COLLECTOR_HTTP_TIMEOUT` | 5s，可设 1s–30s |
| `COLLECTOR_MAX_PENDING` | 1440，可设 1–10080 |

Linux 安装程序另以运行锁与 55 秒进程组截止时间保护 cron；Windows 计划任务调用固定 cmd wrapper，新程序自身有总截止时间。

## 测试与打包

```sh
go test -race ./...
scripts/package.sh
```

打包输出 `dist/linux-amd64.tar.gz`、`dist/windows-amd64.zip` 和 SHA256SUMS，匹配 agent 的现有下载命名。只包含可执行程序和原仓库工具文件，不包含 .env、pending 或工作区文件。此仓库现有工具二进制仅覆盖 amd64，未声称支持其他架构。`COLLECTOR_BUILD_VERSION` 可指定构建版本。

上线顺序：先更新支持多 sample_id/nullable/组时间的主控，再更新 Go 接收器和 agent 同步服务，最后发布主机包并更新安装脚本。已有主机不会仅因替换下载包而自动升级。先做少量 Linux/Windows 实机灰度，重点验证 RAID、传感器权限与任务执行；Windows 目前有交叉编译和协议/解析测试，不能替代实机验证。
