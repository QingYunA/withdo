# Android 同步服务（首个可运行版本）

`adhd-sync` 是独立 Go 服务，使用专用 SQLite 存储手机的项目、任务和追加式报告。手机可离线编辑，再向这台服务同步。服务部署在持续在线的机器后，手机同步不依赖桌面电脑开机。

**当前边界：这是单个所有者的开发 / 自托管服务。** 所有持有同一令牌的设备共享同一份数据；没有 OIDC、用户隔离、账号删除或持续桌面数据库同步。现有桌面、CLI、MCP 数据可由下述命令显式导入快照，不会自动出现在手机里。`internal/domain` 目前只复用状态与优先级校验，服务不调用桌面业务事务，也不会迁移其数据库。

## 启动

仓库根目录运行，使用仓库要求的 Go 版本：

```sh
export ADHD_SYNC_TOKEN="$(openssl rand -hex 32)"
go run ./cmd/adhd-sync -db ./data/mobile-sync.db
```

默认监听 `127.0.0.1:8787`。`ADHD_SYNC_TOKEN` 必须显式设置且非空；也可用 `-token`，但环境变量可避免令牌出现在命令行参数里。不要把令牌写进仓库、URL 或公开日志。长期运行时通过自己的服务管理工具保存令牌，重启时继续使用同一个令牌和数据库路径。

```sh
curl http://127.0.0.1:8787/healthz
curl -H "Authorization: Bearer $ADHD_SYNC_TOKEN" \
  http://127.0.0.1:8787/v1/sync/snapshot
```

第一条返回 `{"status":"ok"}`，第二条在空库返回非空 `epoch`、`cursor: 0` 和空 `records`。

- Android 官方模拟器在手机设置中填 `http://10.0.2.2:8787`，令牌填相同值。`10.0.2.2` 是模拟器访问宿主机回环地址的入口。
- 真机访问局域网时，服务用 `-listen 0.0.0.0:8787` 启动，手机填写电脑的局域网 IP，例如 `http://192.168.1.20:8787`。防火墙需允许连接。手机自己的 `localhost` 指向手机。
- 远程自托管使用 HTTPS 入口代理到服务。该程序只提供 HTTP，不会创建云资源、申请证书或开放公网端口。开发 HTTP 地址需要 Android 构建允许明文流量；正式发布不能把开发地址写死。

按 Ctrl-C 或发送 SIGTERM 会停止接收请求，最多等待 10 秒完成在途请求，再关闭数据库。可用 `go build ./cmd/adhd-sync` 生成本地可执行文件。

## 把既有桌面数据导入手机

这是用户主动执行的一次快照导入。先停止所有使用源库的桌面、CLI、MCP 进程，再取得已经 checkpoint 的一致 SQLite 备份；导入时让源备份保持静止。不要复制一个仍在写入、依赖 WAL 的主文件。命令不会读取任何默认用户数据库，必须明确指定 `-source`：

```sh
go run ./cmd/adhd-sync import-desktop \
  -source /path/to/checkpointed-desktop.db \
  -source-id yunqing-mac-primary \
  -url http://127.0.0.1:8787
```

导入命令使用相同的 `ADHD_SYNC_TOKEN`，经普通鉴权 push API 上传项目、父子任务和已有 Agent 报告。随后在 Android 里同步，即可查看这些任务与报告。仓库绝对路径、milestone、GitHub 轮询缓存等手机契约未包含的字段不会导入；源数据库没有改动。

`-source-id` 是这份桌面数据库的稳定身份。同一库的后续备份一直使用同一个值；不同的独立桌面库必须使用不同值。项目 / 任务 / 报告的整数 ID 按此身份稳定映射成 UUID，父子与项目关系一起映射。操作 ID 还包含内容哈希，因此重复导入相同快照不会重复创建报告或记录。

输出 JSON 包含 `accepted/conflicts/rejected/issues`。`accepted` 包括以前已经保存的幂等回执，不表示覆盖当前远端内容。已有手机编辑会保留；后来桌面修改产生的新内容仍以 `baseVersion: 0` 提交，遇到现存记录会报告 `conflict`，命令退出码为 1，等待显式协调。它不会拉取新版本再自动覆盖手机编辑，也不会把桌面上物理删除的记录转成云端删除。网络中断后的部分成功可用同一命令安全重试。

源文件以 SQLite `mode=ro&immutable=1` 打开，不调用桌面 `store.Open` 或迁移；检查发现非空 WAL 或读取期间源文件变化会拒绝导入。这个边界避免只读打开仍创建 / 修改源库 sidecar，也避免忽略未 checkpoint 的近期数据。请使用停止写入后的干净备份；这条命令不提供持续或双向桌面复制。

## 协议

JSON 类型以 [`mobile/src/model.ts`](../mobile/src/model.ts) 和 [Android 契约](plans/m7-android-contract.md) 为准，字段为 camelCase。所有实体、操作与设备 ID 必须是小写带连字符 UUID；`baseVersion` 是 JavaScript 安全整数范围内的非负整数。新增实体以 `baseVersion: 0` 提交，时间为带时区的 RFC 3339 字符串。

| 接口 | 行为 |
| --- | --- |
| `GET /healthz` | 无需鉴权的数据库就绪检查，不返回配置与凭据 |
| `POST /v1/sync/push` | `{deviceId, mutations}` → `{results}` |
| `GET /v1/sync/pull?cursor=0&limit=100&epoch=...` | 按服务端序号拉取变化，返回 `epoch/cursor/changes/hasMore` |
| `GET /v1/sync/snapshot` | 在同一个 SQLite 读事务中获取 `epoch/cursor/records`，包含 tombstone |

后三个接口必须带 `Authorization: Bearer <token>`。数据响应使用 `Cache-Control: no-store`。服务不会执行收到的分支名、仓库路径或 PR URL。

每个操作单独原子提交实体、版本、变化日志和幂等回执。已经提交但丢失响应的操作可原样重试；不会再次增加版本或变化序号。同一个操作 ID 改内容、设备 ID 或 `baseVersion` 返回 `rejected / mutation_payload_mismatch`。冲突和拒绝回执也会保存，修正操作或用户解决冲突时必须使用新的操作 ID。

当 `baseVersion` 与服务端版本不同，返回 `conflict` 和当前规范记录；不存在记录时 `record: null`。同批其他独立操作继续处理。批次中的多条记录**不具有整体原子性**；父任务完成与子任务完成必须分别校验版本，UI 需要保留其中的冲突，不能声称整个集合一次性成功。

项目必须先于引用它的任务提交，父任务必须先于子任务提交。子任务记录需要明确提供与父任务相同的 `projectId`；客户端负责继承父任务项目。服务拒绝不存在 / 已删除的引用、自身或循环父子关系、超过 100 层的父链，以及会使子任务项目不一致的父任务项目变更。

删除使用 `data: null` 生成版本化 tombstone。项目仍有任务、任务仍有子任务时返回 `dependent_tasks_exist`，不自动级联删除。客户端应先提交子任务删除或解除父子关系并收到确认，再提交父任务删除。已有报告永远保留；报告只能创建，不能覆盖或删除，也不能新增到已经删除的任务。父任务进入终态不会在服务端暗中关闭子任务，客户端显式提交子任务状态变更。完整桌面领域行为与涉及集合的原子命令留给后续协议 / 桌面同步阶段。

离线旧编辑遇到 tombstone 会产生版本冲突，不会静默复活记录。用户明确选择保留本地内容后，可用新的操作 ID 和 tombstone 当前版本提交恢复；这会生成新的版本和变化记录。首版不压缩变化日志、tombstone 或幂等回执。

## 上限与错误

- push 请求体最大 1 MiB，每批最多 100 个操作，每条 `data` 最大 64 KiB；任务备注上限 50,000 字节。
- pull 默认 100 条，每页最多 200 条。`cursor` 必须显式提供且非负；客户端沿返回游标继续拉取直到 `hasMore: false`。
- snapshot 上限 10,000 条记录 / 约 8 MiB，达到上限返回 `413 / snapshot_too_large`，不会截断数据。超出此首版容量前需增加分页快照协议。
- JSON / 参数错误为 `400`，缺少或错误令牌为 `401`，请求过大为 `413`，内部失败为 `500`。响应为 `{"error":"<code>"}`，不包含 SQL、请求内容或令牌。
- 游标超过服务端当前最大序号，或已知 `epoch` 不一致，返回 `409 / resnapshot_required`。客户端保留本地待提交编辑，重新取 snapshot 后再重放，不能清空 outbox。
- 每个操作的校验错误在 HTTP 200 的 `results` 中返回 `rejected`，例如 `invalid_task`、`project_not_found`、`parent_not_found`、`parent_project_mismatch`、`invalid_parent`、`dependent_tasks_exist`、`reports_are_append_only`、`created_at_is_immutable`。

## 数据持久化与备份恢复

数据库路径必须专用于此服务。若检测到桌面或其他应用的表，启动会拒绝，以避免误改既有数据库。正常重启保留 `epoch`、变化序号、tombstone 和所有操作回执。

备份时先停止服务再复制数据库文件，或使用 SQLite 的一致性备份接口；不要在写入时仅复制主文件而遗漏 WAL。恢复备份后，首次启动加 `-reset-epoch`：

```sh
go run ./cmd/adhd-sync -db ./data/mobile-sync.db -reset-epoch
```

这会保留恢复后的实体和操作回执，并生成新的 epoch，让所有设备重新取 snapshot。下一次正常启动应省略该参数。直接复用备份的旧 epoch 无法可靠识别服务端序号回退后又增长的情况。

## 自动化验证

```sh
go test -race ./internal/mobilesync ./cmd/adhd-sync
```

测试覆盖鉴权、畸形与超限请求、字段和关系校验、提交后丢响应并重启重试、操作 ID 内容不一致、版本冲突后独立操作继续、分页增量、tombstone、防止旧编辑复活、追加式报告、子项优先删除、并发写入的一致快照、持久重开和恢复备份后的 epoch 变化。桌面导入使用既有 store / domain 创建临时源库，验证文件与 schema 不变、UUID / 关系 / 报告映射、重复导入、手机修改保护、源变更冲突与源删除不传播。测试通过不能替代 Android 真机安装、系统通知投递或远程部署验收。
