# 工具包开发规范

工具中心按“一个工具一个包”的方式组织。当前 `builtin` 目录存放系统内置工具，后续上传安装的 zip 包也应保持同样结构。

## 目录结构

```text
tool.zip
├─ manifest.json
├─ frontend/
│  ├─ index.html
│  └─ assets/
├─ migrations/
│  ├─ install.sql
│  └─ uninstall.sql
└─ README.md
```

内置工具可以没有独立 `frontend` 目录，通过 `entryType: "native"` 和 `panelKey` 挂载到主系统已有 Vue 面板。

## manifest.json

```json
{
  "toolId": "delivery-json-extract",
  "toolName": "发货 JSON 提取",
  "version": "1.0.0",
  "toolDesc": "解析发货单 JSON，支持查询、分页和 Excel 导出。",
  "toolIcon": "file-json",
  "toolCategory": "数据工具",
  "toolStatus": "active",
  "packageType": "builtin",
  "entryType": "native",
  "entryPath": "",
  "panelKey": "delivery-json-extract",
  "isRecommended": false,
  "isRemovable": false,
  "sortOrder": 15,
  "permissions": ["tools:view", "tools:manage"]
}
```

## 字段说明

- `toolId`: 全局唯一工具 ID，建议使用 kebab-case。
- `version`: 语义化版本号。
- `entryType`: `native` 表示主系统内置面板，`iframe` 表示独立前端入口。
- `panelKey`: `native` 工具对应的主系统面板标识。
- `entryPath`: `iframe` 工具的入口地址。
- `toolStatus`: `active` 可用，`paused` 维护中，`planning` 即将上线。
- `permissions`: 工具需要声明的权限点，安装流程会据此维护权限。

## 当前内置工具

- `product-research`: 商品采集
- `delivery-json-extract`: 发货 JSON 提取
