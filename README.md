# 🚀 Enterprise Go Backend Framework

**面向 SaaS / 多租户 / 高并发场景的企业级 Golang 后端框架**

本框架采用 **模块化 + 领域分层 + 高性能中间件 + 工具库内聚**的架构思想，为企业级项目提供稳定基础设施。

## 📦 Features（特性）

### ✅ 企业级架构分层

-   `app/` ------ 业务主目录
    -   `handler`（Controller 层）\
    -   `service`（业务层）\
    -   `dao`（数据访问层）\
    -   `model`（实体/DTO 定义）\
    -   `router`（路由层）\
    -   `middleware`（中间件）
-   `internal/` ------ 纯内部模块
    -   `logx`（Zap 批量日志）\
    -   `database`（数据库组件）\
    -   `retryx`（重试框架）\
    -   `i18nx`（国际化）\
    -   `filex`（文件工具）\
-   `cmd/` ------ CLI 工具（如 migrate）\
-   `migrations/` ------ 结构化数据库迁移
-   `config/` ------ 配置中心（支持热加载）

## 🏗 Architecture（架构图）

                            ┌─────────────────────────┐
                            │        API Client        │
                            └─────────────┬───────────┘
                                          │
                                          ▼
                              ┌────────────────────┐
                              │      Router        │
                              └─────────┬──────────┘
                                        │ middleware
                                        ▼
                            ┌─────────────────────────┐
                            │   Handler (Controller)  │
                            └─────────────┬───────────┘
                                          │
                                          ▼
                         ┌─────────────────────────────────┐
                         │          Service Layer           │
                         └────────────────┬────────────────┘
                                          │
                                          ▼
                               ┌──────────────────┐
                               │        DAO        │
                               └─────────┬────────┘
                                         │
                                         ▼
                               ┌──────────────────┐
                               │     Database     │
                               │   (GORM / SQL)   │
                               └──────────────────┘

## ⚙️ Built-in Components（内置能力）

### 🔥 1. 高性能日志系统

-   Zap + 自研 BatchWriter（减少 syscalls）
-   支持 JSON / 本地色彩日志
-   支持脱敏（手机号、身份证、卡号）
-   支持 GORM Logger 集成

### 🔥 2. 数据库组件（MySQL）

-   多环境自动连接（Prod/Test/Dev）
-   预编译语句（PrepareStmt）
-   禁止默认事务（大幅提升性能）
-   慢查询上报

### 🔥 3. 配置模块

-   TOML 格式\
-   环境隔离（prod.toml / test.toml）\
-   配置结构体自动映射

### 🔥 4. 国际化 i18nx

-   自动加载语言包\
-   支持链式语言回退

### 🔥 5. 重试组件 retryx

-   指数退避\
-   可自定义重试策略

### 🔥 6. migrations

-   CLI：

```{=html}
<!-- -->
```
    go run cmd/migrate.go up
    go run cmd/migrate.go down

### 📊 Performance（性能）
- Zap 批量日志：减少 40%~60% IO
- GORM 预编译 stmt：+15% QPS
- 禁用默认事务：+25% 写入性能
- Router + Middleware 层全链路可监控

### 🎯 适用场景
- SaaS 多租户系统
- B 端行政系统
- 支付/订单系统
- 中台服务（用户 / 权限 / 账单）
- 高并发 API 服务


