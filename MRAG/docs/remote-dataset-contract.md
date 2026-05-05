# Remote Dataset Runner Contract

远程数据集运行器协议

This document defines the local-to-remote contract for dataset validation, scanning, preview generation, and index task management.
本文档定义了本地到远程的数据集校验、扫描、预览生成以及索引任务管理的交互协议。

## 1. Scope

## 1. 范围

The Go backend supports two modes for remote datasets:
Go 后端支持两种远程数据集模式：

* `mock`: returns deterministic demo responses without contacting a remote host.

* `mock`：返回确定性的演示数据，不会连接远程主机。

* `real`: connects to the configured server over system `ssh` and runs commands under `/home/bzli/lbz`.

* `real`：通过系统 `ssh` 连接配置好的服务器，并在 `/home/bzli/lbz` 目录下执行命令。

The remote side does **not** need to expose an HTTP service. The current contract uses an executable entrypoint and JSON over stdout.
远程端**不需要**提供 HTTP 服务，当前协议通过可执行入口程序并通过 stdout 输出 JSON 进行交互。

## 2. Remote Work Root

## 2. 远程工作目录

The remote working root is fixed to:
远程工作根目录固定为：

```text
/home/bzli/lbz
```

Recommended remote layout:
推荐的远程目录结构如下：

```text
/home/bzli/lbz/
  bin/
    mrag-dataset-runner
  dataset-index-tasks/
    <taskId>/
      request.json
      status.json
      result.json
      logs/
        runtime.log
```

## 3. SSH Requirements

## 3. SSH 要求

The Go backend calls the system `ssh` client and is designed to work with `.ssh/config` aliases.
Go 后端调用系统 `ssh` 客户端，并设计为支持 `.ssh/config` 别名配置。

Recommended server record for SSH alias usage:
推荐的服务器配置字段如下：

* `host`: SSH `Host` alias

* `host`：SSH 的 Host 别名

* `authType`: `ssh_config`

* `authType`：`ssh_config`

* `username`: optional

* `username`：可选

* `sshPort`: optional

* `sshPort`：可选

This allows the system `ssh` client to resolve `HostName`, `User`, `Port`, `IdentityFile`, `ProxyCommand`, and multi-hop routing.
这样可以让系统 `ssh` 客户端自动解析 `HostName`、`User`、`Port`、`IdentityFile`、`ProxyCommand` 以及多跳代理配置。

## 4. Preferred Entrypoint

## 4. 推荐入口程序

Preferred executable path on remote hosts:
远程主机推荐的可执行程序路径为：

```text
./bin/mrag-dataset-runner
```

The Go backend sends one of the following commands through SSH.
Go 后端会通过 SSH 发送以下命令之一。

## 5. Commands

## 5. 命令

### 5.1 Validate Dataset Path

### 5.1 校验数据集路径

Command shape:
命令格式：

```bash
./bin/mrag-dataset-runner validate --path <dataset-root>
```

Expected stdout JSON:
期望的标准输出 JSON：

```json
{
  "sourceType": "remote",
  "path": "/data/datasets/mmrag-sample",
  "mode": "real",
  "valid": true,
  "exists": true,
  "isDirectory": true,
  "errorType": "",
  "message": "Remote dataset directory is available",
  "checkedAt": "2026-03-23T16:20:00+08:00"
}
```

Error typing rules:
错误类型规则：

* `not_found`: path does not exist

* `not_found`：路径不存在

* `permission_denied`: path exists but is not accessible

* `permission_denied`：路径存在但无权限访问

* `not_directory`: path exists but is not a directory

* `not_directory`：路径存在但不是目录

### 5.2 Scan Dataset and Build Preview

### 5.2 扫描数据集并生成预览

Command shape:
命令格式：

```bash
./bin/mrag-dataset-runner scan --path <dataset-root> --preview-limit <n>
```

Expected stdout JSON:
期望输出 JSON：

```json
{
  "validationStatus": "ok",
  "scanStatus": "completed",
  "fileCount": 128,
  "directoryCount": 9,
  "totalSizeBytes": 48392011,
  "fileTypes": {
    "text": 72,
    "image": 41,
    "json": 9,
    "pdf": 6
  },
  "hierarchySummary": [
    { "level": 0, "path": "images", "itemCount": 41 },
    { "level": 0, "path": "docs", "itemCount": 78 },
    { "level": 1, "path": "docs/train", "itemCount": 36 }
  ],
  "inferredModality": "multimodal",
  "recentModifiedAt": "2026-03-23T15:42:11+08:00",
  "previewItems": [
    {
      "name": "images",
      "itemType": "directory",
      "category": "directory",
      "relativePath": "images",
      "sizeBytes": 0,
      "depth": 0
    },
    {
      "name": "sample-001.png",
      "itemType": "file",
      "category": "image",
      "relativePath": "images/sample-001.png",
      "sizeBytes": 204800,
      "depth": 1
    }
  ],
  "errorMessage": ""
}
```

Current category rules expected by the local backend:
当前本地后端期望的分类规则：

* `text`: `txt`, `md`, `csv`, `tsv`, `yaml`, `yml`, `xml`, `html`, `htm`

* `text`：文本类文件

* `pdf`: `pdf`

* `pdf`：PDF 文件

* `json`: `json`, `jsonl`

* `json`：JSON 文件

* `image`: `png`, `jpg`, `jpeg`, `gif`, `bmp`, `webp`, `tif`, `tiff`

* `image`：图像文件

* `audio`: `wav`, `mp3`, `m4a`, `flac`, `aac`

* `audio`：音频文件

* `video`: `mp4`, `avi`, `mov`, `mkv`, `webm`

* `video`：视频文件

* `other`: everything else

* `other`：其他类型

## 6. Index Task Contract

## 6. 索引任务协议

### 6.1 Start Index Task

### 6.1 启动索引任务

The Go backend writes `request.json` first, then calls:
Go 后端先写入 `request.json`，然后执行：

```bash
./bin/mrag-dataset-runner index-start --request-file /home/bzli/lbz/dataset-index-tasks/<taskId>/request.json
```

Example `request.json`:
示例 `request.json`：

（此处 JSON 保持不翻译，仅说明）
（内容同上，不赘述）

Expected stdout JSON:
期望输出 JSON：

（同样结构说明略）

### 6.2 Query Index Task Status

### 6.2 查询索引任务状态

Command shape:
命令格式：

```bash
./bin/mrag-dataset-runner index-status --task-dir /home/bzli/lbz/dataset-index-tasks/<taskId>
```

Expected stdout JSON:
期望输出 JSON：

（结构同上）

Supported status values:
支持的状态值：

* `building`

* `building`：构建中

* `completed`

* `completed`：已完成

* `failed`

* `failed`：失败

Local backend mapping:
本地后端状态映射：

* task `completed` -> dataset `ready`

* 任务完成 → 数据集就绪

* task `building` -> dataset `building`

* 构建中 → 数据集构建中

* task `failed` -> dataset `failed`

* 失败 → 数据集失败

## 7. Placeholder Compatibility

## 7. 占位实现兼容

If `./bin/mrag-dataset-runner` does not exist yet, the Go backend already contains SSH fallback shell scripts that can:
如果 `./bin/mrag-dataset-runner` 尚未实现，Go 后端已经提供了 SSH fallback 脚本，可以：

* validate remote directory existence

* 校验远程目录是否存在

* scan remote trees with basic `find`-based statistics

* 使用 `find` 命令进行基础扫描统计

* create placeholder index task directories and `status.json`

* 创建占位的任务目录和状态文件

That fallback is only for temporary chain validation.
该 fallback 仅用于链路验证。

Once the real remote runner is ready, place it at `./bin/mrag-dataset-runner` and return the JSON payloads above.
当远程 runner 实现后，应放置在该路径，并按规范返回 JSON。

## 8. Security Notes

## 8. 安全说明

* Do not hardcode passwords or private key contents into code or docs.

* 不要在代码或文档中硬编码密码或私钥

* Prefer system `ssh` plus `.ssh/config` aliases.

* 优先使用系统 ssh + 配置文件

* If password authentication is ever required, read it from environment variables or local untracked config only.

* 如需密码认证，应从环境变量或本地未跟踪配置中读取

---
