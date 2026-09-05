# Xboard API 文档

**Base URL：** `http://你的域名`

---

## 接口加密与认证机制

### 传输层
所有接口**仅依赖 HTTPS 传输加密**，请求体/响应体本身为明文 JSON，无额外对称加密。

### 四种认证方式

#### 1. 用户 / 管理员 —— Laravel Sanctum Bearer Token

登录或注册后，服务器返回：

```json
{
  "data": {
    "token": "用户订阅token（UUID格式）",
    "auth_data": "Bearer xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "is_admin": false
  }
}
```

- `auth_data` 就是 Sanctum 颁发的个人访问令牌，有效期 **1 年**
- 后续所有需要登录的请求，在 Header 中携带：

```
Authorization: Bearer xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

> 注意：`token`（订阅 token）和 `auth_data`（API 令牌）是两个不同的字段，不要混用。

#### 2. 客户端订阅 —— 用户 Token（URL 参数）

用户的 `token` 字段（UUID 格式），通过 URL 路径或 query 参数传递：

```
GET /s/{token}
GET /api/v1/client/subscribe?token={token}
```

无需 Authorization Header，token 即身份凭证。

#### 3. 节点服务端 —— 全局 Server Token（Query 参数）

所有节点共享同一个 `server_token`（在管理面板 → 系统配置中设置），以 query 参数传递：

```
GET /api/v1/server/UniProxy/user?token={server_token}&node_id={id}
```

无签名、无时间戳，token 匹配即通过，**请务必通过 HTTPS 传输**。

#### 4. 机器节点 —— Machine Token（Query 参数）

每台机器有独立 token，存储于数据库，通过 `machine_id` + `token` 联合认证：

```
GET /api/v2/server/config?machine_id={id}&token={machine_token}&node_id={node_id}
```

---

## 认证方式汇总

| 场景 | Header / 参数 | 来源 |
|------|--------------|------|
| 用户/管理员 API | `Authorization: Bearer {auth_data}` | 登录接口返回 |
| 订阅/客户端 | URL 路径 `/s/{token}` | 用户信息中的 `token` 字段 |
| 节点服务端 | `?token={server_token}&node_id={id}` | 管理面板系统配置 |
| 机器节点 | `?machine_id={id}&token={machine_token}` | 管理面板机器管理 |

---

## 通用响应格式

```json
{ "data": ..., "code": 0, "message": "success" }
// 错误
{ "message": "错误说明", "code": 错误码 }
```

---

## 一、订阅接口（客户端对接）

### 获取订阅
```
GET /s/{token}
```

Hiddify / sing-box / Clash 等客户端直接请求此 URL，Xboard 根据 User-Agent 自动返回对应格式配置。

| 参数 | 类型 | 说明 |
|------|------|------|
| `types` | string（可选）| 过滤协议类型，如 `vless,trojan`，默认 all |
| `filter` | string（可选）| 按节点名称/标签过滤关键词 |
| `flag` | string（可选）| 手动指定客户端标识（默认读 User-Agent）|

**已支持的客户端标识（flag）：**

| 客户端 | flag 关键词 |
|--------|------------|
| Hiddify | `hiddify` |
| sing-box | `sing-box` / `sfm` |
| Clash Meta | `clash-meta` / `mihomo` |
| Clash | `clash` |
| Shadowrocket | `shadowrocket` |
| QuantumultX | `quantumult%20x` |
| Surge | `surge` |
| Stash | `stash` |
| Loon | `loon` |
| 通用（base64）| 其他 |

---

## 二、Guest 接口（无需认证）

### 获取站点配置
```
GET /api/v1/guest/comm/config
```
返回：`tos_url`、`is_email_verify`、`is_invite_force`、`captcha_type`、`logo` 等

### 获取套餐列表
```
GET /api/v1/guest/plan/fetch
```

### 支付回调
```
GET|POST /api/v1/guest/payment/notify/{method}/{uuid}
```

---

## 三、Passport 接口（注册/登录）

### 发送邮箱验证码
```
POST /api/v1/passport/comm/sendEmailVerify
```
| 参数 | 必填 | 说明 |
|------|------|------|
| `email` | ✓ | 邮箱地址 |

### 注册
```
POST /api/v1/passport/auth/register
```
| 参数 | 必填 | 说明 |
|------|------|------|
| `email` | ✓ | 邮箱 |
| `password` | ✓ | 至少 8 位 |
| `invite_code` | - | 邀请码（站点开启强制邀请时必填）|
| `email_code` | - | 邮箱验证码（开启邮箱验证时必填）|

返回：`{ token, auth_data }`

### 登录
```
POST /api/v1/passport/auth/login
```
| 参数 | 必填 |
|------|------|
| `email` | ✓ |
| `password` | ✓ |

返回：`{ token, auth_data }`

### 忘记密码
```
POST /api/v1/passport/auth/forget
```
| 参数 | 必填 |
|------|------|
| `email` | ✓ |
| `password` | ✓（新密码，至少 8 位）|
| `email_code` | ✓ |

---

## 四、User 接口（需 Bearer Token）

### 用户信息
```
GET /api/v1/user/info
```
返回：`email`、`uuid`、`transfer_enable`、`expired_at`、`balance`、`plan_id` 等

### 订阅信息
```
GET /api/v1/user/getSubscribe
```
返回：`subscribe_url`、`token`、`expired_at`、`u`（上行）、`d`（下行）、`transfer_enable`、`reset_day`

### 修改密码
```
POST /api/v1/user/changePassword
```
| 参数 | 必填 |
|------|------|
| `old_password` | ✓ |
| `new_password` | ✓ |

### 注销账户
```
POST /api/v1/user/cancelAccount
```
| 参数 | 必填 | 说明 |
|------|------|------|
| `password` | ✓ | 当前登录密码，用于二次确认 |

说明：软删除，账户会被禁用并匿名化（邮箱/Telegram/备注被清空、订阅 token 失效），历史订单/工单保留。管理员账户不可通过此接口注销，需联系其他管理员处理。成功后会清空该用户所有 `auth_data`（相当于全端登出），客户端应在收到成功响应后清除本地保存的 `auth_data` 并跳转登录页。

失败响应：
- `400` + `Incorrect password`：密码错误
- `400` + `Admin accounts cannot cancel themselves...`：管理员账户
- `500` + `Cancel failed`：服务端异常

### 重置订阅 Token
```
GET /api/v1/user/resetSecurity
```

### 获取服务器列表
```
GET /api/v1/user/server/fetch
```
返回：可用节点数组（支持 ETag 304 缓存）

### 流量日志
```
GET /api/v1/user/stat/getTrafficLog
```

### 获取套餐
```
GET /api/v1/user/plan/fetch
```

### 获取公告
```
GET /api/v1/user/notice/fetch
```

### 知识库文章
```
GET /api/v1/user/knowledge/fetch
```
| 参数 | 说明 |
|------|------|
| `id` | 文章 ID（可选）|
| `keyword` | 搜索关键词（可选）|

---

## 五、Order 接口

### 创建订单
```
POST /api/v1/user/order/save
```
| 参数 | 必填 | 说明 |
|------|------|------|
| `plan_id` | ✓ | 套餐 ID |
| `period` | ✓ | 周期，如 `month_1`、`year_1` |
| `coupon_code` | - | 优惠券码 |

返回：`trade_no`

### 结账
```
POST /api/v1/user/order/checkout
```
| 参数 | 必填 |
|------|------|
| `trade_no` | ✓ |
| `method` | ✓（支付方式 ID）|

### 查询订单状态
```
GET /api/v1/user/order/check?trade_no=xxx
```
返回：`0`=待付款，`1`=开通中，`2`=已取消，`3`=已完成

### 取消订单
```
POST /api/v1/user/order/cancel
```
| 参数 | 必填 |
|------|------|
| `trade_no` | ✓ |

### 获取支付方式
```
GET /api/v1/user/order/getPaymentMethod
```

### 订单列表
```
GET /api/v1/user/order/fetch
```

---

## 六、Invite 邀请接口

### 获取邀请码及统计
```
GET /api/v1/user/invite/fetch
```
返回：`codes`（邀请码列表）、`stat`（邀请人数、佣金等）

### 生成邀请码
```
GET /api/v1/user/invite/save
```

### 佣金明细
```
GET /api/v1/user/invite/details
```

---

## 七、Ticket 工单接口

### 工单列表
```
GET /api/v1/user/ticket/fetch
```

### 创建工单
```
POST /api/v1/user/ticket/save
```
| 参数 | 必填 | 说明 |
|------|------|------|
| `subject` | ✓ | 标题 |
| `level` | ✓ | 优先级 0/1/2 |
| `message` | ✓ | 内容 |

### 回复工单
```
POST /api/v1/user/ticket/reply
```
| 参数 | 必填 |
|------|------|
| `id` | ✓ |
| `message` | ✓ |

### 关闭工单
```
POST /api/v1/user/ticket/close
```

---

## 八、Server 接口（节点端调用）

> 认证：所有请求需带 `?token={server_token}&node_id={id}`
> server_token 在管理面板 → 系统配置中查看

### 拉取用户列表
```
GET /api/v1/server/UniProxy/user
```
返回：用户数组（含 uuid、流量限制、速率限制等），支持 ETag 304

### 拉取节点配置
```
GET /api/v1/server/UniProxy/config
```
返回：节点配置 + `base_config`（push_interval / pull_interval）

### 上报流量
```
POST /api/v1/server/UniProxy/push
```
Body（JSON 数组）：
```json
[
  { "user_id": 1, "u": 1024, "d": 2048 }
]
```

### 上报在线用户
```
POST /api/v1/server/UniProxy/alive
```
Body：
```json
{ "data": [1, 2, 3] }
```

### 获取在线列表（设备数限制用）
```
GET /api/v1/server/UniProxy/alivelist
```

### 上报节点状态
```
POST /api/v1/server/UniProxy/status
```
```json
{
  "cpu": 30,
  "mem": { "total": 4000000000, "used": 1000000000 },
  "swap": { "total": 0, "used": 0 },
  "disk": { "total": 50000000000, "used": 10000000000 }
}
```

---

## 九、Admin 接口（管理员）

> 管理面板路径：`/api/v2/{admin_path}/`（安装时生成的随机路径）
> 本站管理路径：`/f1b8d58c`

| 模块 | 路径前缀 |
|------|---------|
| 用户管理 | `POST /api/v2/admin/user/fetch` |
| 订单管理 | `POST /api/v2/admin/order/fetch` |
| 套餐管理 | `GET /api/v2/admin/plan/fetch` |
| 节点管理 | `GET /api/v2/admin/server/manage/getNodes` |
| 优惠券 | `GET /api/v2/admin/coupon/fetch` |
| 工单 | `GET /api/v2/admin/ticket/fetch` |
| 系统配置 | `GET /api/v2/admin/config/fetch` |
| 统计数据 | `GET /api/v2/admin/stat/getOverride` |
| 插件管理 | `GET /api/v2/admin/plugin/getPlugins` |
| 主题管理 | `GET /api/v2/admin/theme/getThemes` |

---

## Hiddify 对接说明

只需将订阅 URL 填入 Hiddify，无需其他配置：

```
http://你的域名/s/{用户token}
```

Xboard 检测到 Hiddify 的 User-Agent 后，自动返回 sing-box 格式配置。

**支持的协议：** Shadowsocks、Trojan、VMess、VLESS、Hysteria2、TUIC、AnyTLS、SOCKS、HTTP
