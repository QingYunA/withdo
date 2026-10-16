# Android 首版交付与验证记录

日期：2026-10-16。worktree：`codex/android-client`。应用代码提交：`7a3b05c73c33bda969623b80797eb06b791ac97e`。

## 已实现

- Expo SDK 58 / React Native 0.88 Android 客户端，版本 `0.3.0`，应用 ID `dev.adhdtodo.mobile`。
- Mine / Agents、项目创建 / 改名 / 归档、任务与子任务、搜索、日历、提醒、中英文、明暗模式。
- 本地 SQLite、业务变更与 outbox 同事务、杀进程后的数据恢复、断开同步后保留本地数据。
- 独立 Go 同步服务：版本检查、持久化幂等回执、删除标记、一致快照、增量拉取和显式冲突处理。
- 用户主动执行的只读桌面快照导入，包含项目、父子任务和真实 Agent 报告。不会修改源备份，不会静默覆盖手机编辑。

## 安装包

- 文件：`ADHD-Todo-0.3.0-android.apk`，60,402,359 字节，约 58 MiB。
- 同时包含 `arm64-v8a`、`x86_64` 原生库和内嵌 JS bundle，启动不需要 Metro。
- SHA-256：`487ed54185d56c21be6d6f331c2886f5d06a2127f53e29300930d72a321c70cd`。
- 使用测试签名，适合安装验收；尚未配置正式发布密钥或上传 Google Play。
- [原生构建与自动化运行](https://github.com/QingYunA/adhd-todo/actions/runs/38852900815)；[草稿 PR #2](https://github.com/QingYunA/adhd-todo/pull/2)。

## 验证结果

| 检查 | 结果与边界 |
| --- | --- |
| 全仓库 Go 竞态测试 | `go test -race ./...` 通过，包含现有桌面包 |
| Go 静态检查 | `go vet ./...` 通过 |
| 桌面翻译目录检查 | 344 条 en / zh-Hans 文案通过 |
| 手机类型与 lint | TypeScript、ESLint 均通过 |
| Expo 配置与依赖 | `expo-doctor` 20/20；SDK 兼容性、peer 检查通过 |
| SQLite / UI 行为测试 | 本地 36 项通过；真实 Go HTTP 联调用例在 CI 中启用并通过，共 37 项 |
| 原生 SQLite 连接差异 | 新连接的外键检查、延迟约束、回滚和重新打开专项测试通过；构建启用 `SQLITE_DEFAULT_FOREIGN_KEYS=1` |
| Android 原生构建 | CI `assembleRelease` 成功；本地 APK ZIP 完整性、ABI、内嵌 bundle 与 SHA-256 验证通过 |
| Android 模拟器 | Android API 35 / Pixel 6、飞行模式创建任务与笔记、杀进程重开、日历切换、明暗模式重开等断言通过 |
| 真机 | 尚未验证；通知投递、电池限制、中文 IME、TalkBack 与升级行为需在实际设备验收 |

初轮模拟器镜像下载出现坏 ZIP，重跑后成功；应用检查与原生构建未受影响。首轮通过的模拟器日志包含全部断言，但截图上传因隐藏目录被过滤；归档配置已修正，并在[补归档运行](https://github.com/QingYunA/adhd-todo/actions/runs/38856926109)中验证手机源码完全一致后复用同一个 APK。该运行的 checks、apk 和 emulator 全部成功。

[浅色截图](../design/screens/android/mine-light.png)和[深色截图](../design/screens/android/mine-dark.png)来自实际安装的 APK。独立视觉审查对这两个 Mine 状态给出通过结论：未发现裁切、重叠或明显可读性问题，当前可见文字组合对比度不低于 4.59:1。日志中未发现应用 fatal、ANR、JS invariant 或 SQLite 锁异常。该结论不覆盖未截图页面、放大字体、中文 IME、读屏、平板或真机通知。

## 审查发现与修复

两路独立审查分别检查代码标准与需求行为，以下问题均经原始复现和回归测试确认修复：

1. 断开同步清除已确认缓存 → 保留本地数据、版本、游标与待提交操作；同端点重连继续工作。
2. 大笔记使请求超过 1 MiB 并永久重试 → 按完整 UTF-8 JSON 字节分批，保留依赖顺序，非法旧项单独进入冲突。
3. 并发新增 / 保留子任务使父删除卡住 → 显示父删除冲突，明确选择保留远端或解绑当前子任务后继续删除。
4. Agent / 已完成任务图标误完成 → 打开详情，个人活跃任务才允许一键完成；实际事件处理测试覆盖 14 种组合。
5. JSON 大小写别名绕过实体身份校验 → 严格拒绝不匹配的字段名和重复键，覆盖任务、项目、报告与请求外层。

父代理另检查了 Expo 独占事务的新连接行为，并补充构建标志、运行时断言和实际 SQLite 独立连接测试。审查通过只覆盖代码与上述场景，不替代真机验收。

## 后续 M7 工作

当前是 Android 测试首版，未把整个跨端计划标为完成：

- 手机可独立离线使用；跨设备同步需要在电脑之外部署 Go 服务，目前没有创建外部部署。
- 现有桌面库当前是显式快照导入，持续双向桌面 / CLI / MCP 复制与旧库迁移尚未实现。
- 同步使用单个所有者的访问密钥；正式 OIDC、设备撤销与多账号隔离尚未实现。
- 系统本地提醒已实现；远程推送服务、实时后台保证和实际设备投递尚未验收。
- iOS 构建、TestFlight 和双商店正式发布留待后续。

运行与连接方式见 [mobile/README.md](../../mobile/README.md) 和 [Android 同步服务](../android-sync.md)。
