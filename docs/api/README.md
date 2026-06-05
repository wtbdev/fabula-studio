# 叙幕工作室（Fabula Studio）API 文档包

本目录按照 Spec Coding / api.spec.md 的接口规约思路组织：先定义接口契约，再进入编码实现。

## 文档列表

| 文件 | 说明 |
|---|---|
| `00-api.spec.md` | API 总览、统一规范、鉴权、分页、错误格式 |
| `01-auth.api.spec.md` | 用户注册、登录、当前用户、退出登录接口 |
| `02-projects.api.spec.md` | 项目（剧本）增删改查接口 |
| `03-generation.api.spec.md` | 小说转剧本生成接口、生成状态、AI 返回格式 |
| `04-scenes.api.spec.md` | 项目编辑页所需的场次列表、场次详情、保存场次接口 |
| `05-models.schema.md` | 数据模型、数据库表结构、TypeScript 类型定义 |
| `06-error-codes.md` | 通用错误码与业务错误码 |
| `openapi.yaml` | OpenAPI 3.0 草案，方便后续导入接口工具 |

## 今日 MVP 范围

今日需要完成的最小实现：

1. 用户系统：注册、登录、获取当前用户。
2. 项目管理：项目创建、列表、详情、修改、删除。
3. 生成功能：项目调用 AI 生成剧本场次。
4. 项目编辑：左侧场次列表，右侧主编辑器，支持保存当前场次内容。

## 统一约定

- Base URL：`/api`
- 鉴权方式：`Authorization: Bearer <token>`
- 响应格式：`{ code, message, data }`
- 时间格式：ISO 8601 字符串，例如 `2026-06-06T10:00:00.000Z`
- MVP 阶段生成剧本固定扣除 `300` AI 点数。

