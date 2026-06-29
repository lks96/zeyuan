# Temu Tools

Temu Tools 是一个网页版运营工具台，后端使用 Go，前端使用 Vue 3 + Vite + Vuestic UI。

## 项目结构

```text
.
├── backend/   # Go HTTP API、MySQL 连接、数据模型
└── frontend/  # Vue 3 + Vite + Vuestic UI
```

## 多用户与店铺模型

当前数据模型支持：

- 一个系统用户可以关联多个 Temu 店铺
- 一个店铺也可以授权给多个用户
- `admin` 角色可以查看所有用户和所有店铺
- `user` 角色只能查看自己被分配的店铺

核心表：

- `users`
- `shops`
- `user_shops`

建表脚本分两层维护：新库使用完整建库脚本，旧库使用版本化迁移补丁。迁移目录在 [migrations](C:/Users/admin/Documents/temu-tools/backend/migrations/README.md)，完整建库脚本在 [zeyuan_db_full_schema.sql](C:/Users/admin/Documents/temu-tools/backend/scripts/zeyuan_db_full_schema.sql)。

## 数据库配置

后端通过环境变量读取 MySQL 连接信息，不把密码写进源码。先复制示例配置：

```bash
cd backend
cp .env.example .env
```

然后把 `.env` 里的 `DB_PASSWORD` 改成实际密码。你提供的 MySQL 可访问库名是 `runtu_trade`；这个业务是典型关系型数据，多用户、店铺授权、订单/商品后续也都适合 MySQL。后面如果需要本地开发隔离，可以再补一个 Docker MySQL。

执行数据库准备：

```bash
go run ./cmd/dbprepare
```

`dbprepare` 会自动判断当前数据库状态：如果库不存在或是空库，就执行完整建库脚本；如果库内已有表，就执行迁移补丁。迁移器会维护 `schema_migrations` 表，记录已执行文件的版本号和 checksum。已经执行过的 SQL 文件不要直接修改，后续改表、补字段、补初始化权限都新增 `003_xxx.sql` 这类文件。

全新库初始化管理员账号时，需要在 `.env` 或系统环境变量中设置：

```bash
ADMIN_INITIAL_PASSWORD=change_this_admin_password
```

如果以后换成有建库权限的账号，迁移命令会自动创建配置中的数据库。手动创建 SQL 如下：

```sql
CREATE DATABASE temu_tools CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

当前机器可以使用 `E:\tools\activate.ps1` 激活 Go：

```powershell
. E:\tools\activate.ps1
```

## 启动后端

```bash
cd backend
go run ./cmd/server
```

默认监听 `http://localhost:8080`。

开发阶段默认使用 `DEV_USER_ID=1`，也就是初始化脚本里的管理员账号。请求头传 `X-User-ID: 2` 可以模拟普通用户。

`.env.example` 里启用了 `DEV_AUTH_ENABLED=true`，用于本地调试时允许 `X-User-ID`。正式环境不要开启它，登录后使用 Bearer token 即可。

## 启动前端

```bash
cd frontend
npm install
npm run dev
```

默认监听 `http://localhost:5173`，并通过 Vite proxy 请求 `/api` 到 Go 后端。

## 当前页面

- 登录页：管理员和普通用户登录入口
- 仪表盘：后端状态、当前身份、店铺/用户/工具模块概览
- 工具中心：预留工具列表和启用状态
- 店铺管理：展示当前用户可见店铺，管理员可见所有店铺
- 用户管理：管理员查看用户列表
- 系统设置：预留 API 地址、店铺别名和同步间隔配置

## 初始化账号

初始化 SQL 会创建两个账号：

```text
管理员：admin / admin123
普通用户：operator_a / operator123
```

登录接口是 `POST /api/auth/login`。当前阶段使用后端签名的轻量 Bearer token，前端登录后会保存当前用户和 token，并在后续请求中带：

```text
Authorization: Bearer <token>
```

后续正式做权限时建议升级为更完整的 JWT claims 或 HttpOnly session，并把密码哈希换成 bcrypt/Argon2。

## 后台 API

公开接口：

```text
GET  /api/health
POST /api/auth/login
```

登录后可用：

```text
GET /api/me
GET /api/tenant/summary
GET /api/modules
GET /api/shops
GET /api/settings
```

按权限控制的后台接口：

```text
POST   /api/modules
PUT    /api/modules/{id}
DELETE /api/modules/{id}

GET    /api/users
POST   /api/users
PUT    /api/users/{id}
DELETE /api/users/{id}

POST   /api/shops
PUT    /api/shops/{id}
DELETE /api/shops/{id}

GET    /api/users/{id}/shops
POST   /api/users/{id}/shops
DELETE /api/users/{id}/shops/{shopID}

PUT    /api/settings

GET    /api/permissions
GET    /api/roles/{role}/permissions
PUT    /api/roles/{role}/permissions
```

店铺删除目前是软删除：把状态改成 `closed`。用户删除目前是停用：把状态改成 `disabled`。

## API 测试

后端接口集成测试脚本在 [test-api.ps1](C:/Users/admin/Documents/temu-tools/backend/scripts/test-api.ps1)。确保后端正在 `http://localhost:8080` 运行后执行：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File backend\scripts\test-api.ps1
```

测试覆盖：

- 健康检查
- 登录和权限返回
- 普通用户禁止访问用户接口
- 当前用户、仪表盘汇总
- 工具模块增改删
- 系统设置读写
- 权限字典和角色权限查询
- 店铺增改关
- 用户列表、用户店铺分配和移除

## 权限管理

系统使用按钮级权限码控制菜单、页面入口和操作按钮。初始化迁移会创建：

- `permissions`：权限码字典
- `role_permissions`：角色到权限码的授权关系

默认权限：

- `admin`：拥有全部权限
- `user`：拥有 `dashboard:view`、`tools:view`、`shops:view`、`settings:view`

当前权限码：

```text
dashboard:view
tools:view
tools:manage
tasks:create
shops:view
shops:create
shops:update
shops:delete
users:view
users:create
users:update
users:disable
users:assign_shops
settings:view
settings:update
permissions:view
permissions:manage
```

前端会在登录后保存权限码，并在进入后台时调用 `/api/me` 同步最新权限。菜单和按钮都通过同一套权限码判断，例如“用户管理”需要 `users:view`，“保存系统设置”需要 `settings:update`。
