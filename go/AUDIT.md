# 页面显示自查：与 PHP 原版仍有差异的接口清单

方法：同一份数据库副本上同时跑 PHP 原版和 Go 版，对 59 个管理端接口、38 个用户端/客户端接口做字段路径、类型、值的逐项比对；
再用浏览器打开页面核对（页面文字扫描、弹窗、各协议节点表单、翻译键、样式类）。下面是最后一轮仍有差异的接口，以及为什么不影响显示。

## 管理端（28）

| 接口 | 差异 | 结论与依据 |
|---|---|---|
| config/fetch | `app_name` 为 Lets VPN；sing-box 模板末尾换行 | 品牌是有意保留；模板末尾换行已修（trim 后再缩进），与 PHP 逐字节一致 |
| gift-card/codes | 缺嵌套 `template`、`updated_at`；`per_page` 默认 10 | 前端读 `template_name`，请求总是显式传 `per_page` |
| gift-card/templates、usages | `per_page` 默认 10 | 前端显式传 `per_page: 1000` / 页大小 |
| gift-card/types | PHP 多返回数字键 | 前端用固定的 `types.1..4` 文案，不读此接口 |
| payment/fetch | 缺 `created_at/updated_at`；`handling_fee_percent` 为数字 | 前端不读时间；数字更适合渲染 |
| plugin/getPlugins | 缺 `auto_import*` 配置项 | 属于 PHP 里 BalalaSub 插件自带的配置，Go 版没有该插件 |
| server/machine/fetch | `created_at/updated_at` 为字符串 | 机器页只读 `load_status.updated_at`、`last_seen_at` |
| server/machine/history | 数值不同 | 两边测试数据不同，不是结构差异 |
| server/machine/nodes | 422 时缺 `errors` 字段 | 仅参数错误场景，前端只看 `message` |
| server/manage/syncStats | `synced` 数值不同 | 取决于各客户端上报的 revision，数据状态不同 |
| stat/getOverride、getStats、getTrafficRank、getStatUser | PHP 返回字符串数字，Go 返回数字 | 前端按数字格式化，两种都能渲染，数字更稳 |
| stat/getServerYesterdayRank | 数值不同 | PHP 把全部历史相加（PHP 的 bug），Go 按「昨日」 |
| stat/getRanking、getStatRecord | PHP 500 | PHP 自身报错，Go 正常返回 |
| system/getAuditLog | 缺 `updated_at` | 前端只读 `created_at` |
| system/getHorizonFailedJobs | 缺 `current/page_size` | Go 版没有 Horizon，页面按空列表渲染 |
| system/getQueueMasters | PHP 403 / Go 404 | Go 版无 Horizon |
| system/getQueueStats、getSystemStatus | `status/processes/horizon` 值不同 | PHP 的 Horizon 没启动；Go 展示内置调度器状态 |
| ticket/fetch | 用户字段较少 | 前端只读邮箱和 id |
| traffic-reset/logs | `trigger_source_name` 中文 | 有意：页面展示更友好 |
| user/getUserInfoById | 余额为元（PHP 为分） | 前端不调用该接口 |
| order/detail | `plan.sell` 布尔；`user.balance` 单位 | 前端只当真假用；详情不读用户余额 |
| payment/getPaymentForm | PHP 500 | PHP 自身报错 |

## 用户端（3）

| 接口 | 差异 | 结论与依据 |
|---|---|---|
| user/knowledge/getCategory | PHP 500 | PHP 自身报错 |
| user/stat/getTrafficLog | `server_rate` 数字 vs 字符串 | 主题按数字乘/显示，均可渲染 |
| client/app/getConfig | YAML 键顺序不同 | 内容来自同一份 `app.clash.yaml`，键顺序不影响客户端解析 |

## 浏览器实测覆盖

- 管理端：仪表盘、系统设置 15 个分组、支付/插件/主题/公告/知识库/套餐/订单/优惠券/礼品卡/节点/权限组/路由/服务器/用户/工单；
  添加与编辑弹窗；节点弹窗 11 种协议 × 3 个标签页。
- 用户端：仪表盘、购买订阅、节点状态、订单、邀请、个人中心、工单、流量明细、使用文档。
- 检查项：露出的翻译键、`undefined/NaN/[object`、被压窄的输入框、横向溢出、弹窗尺寸与遮罩、缺失的样式类。

## 无法在本环境验证

- Telegram 机器人、SMTP 发信、真实支付网关（无账号）。
- 客户端「导航」页点按钮报 `ERR_CLEARTEXT_NOT_PERMITTED`：客户端 WebView 拒绝 `http://` 页面；
  面板下发的 `nav_page_url` 是 `https://nav.gotab.cn`，需客户端放开明文或外链改用系统浏览器。
