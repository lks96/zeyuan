# Temu Tools Chrome 插件

这个插件用于在 Temu 卖家中心页面捕获接口 JSON，并同步到本项目现有接口：

- 商品采集：`POST /api/tools/product-collection/import-json`
- 发货 JSON 提取：`POST /api/tools/delivery-extractions/import-json`

商品数据会优先识别卖家中心接口：

```text
https://agentseller.temu.com/visage-agent-seller/product/skc/pageQuery
```

页面请求该接口后，插件会缓存响应 JSON，并记录请求里的 `page`、`pageSize`。

发货数据会优先识别卖家中心接口：

```text
https://seller.kuajingmaihuo.com/bgSongbird-api/supplier/deliverGoods/management/pageQueryDeliveryBatch
```

页面请求该接口后，插件会缓存响应 JSON，并记录请求里的 `pageNo`、`pageSize`、`status`，同步时把完整响应提交给后端解析。

## 抓取全部

插件弹窗提供：

- `抓取全部`：复用已捕获的列表请求，在当前卖家中心页面上下文中从第 1 页开始翻页，合并为一份完整 JSON。
- `抓全并同步`：先抓取全部，再提交到本项目接口。

使用前需要先在对应卖家中心列表页触发一次接口请求，例如刷新列表或翻页。插件会复用这次请求里的 `anti-content`、`mallid`、筛选条件、分页大小和当前浏览器登录态。

## 自动检查新品

在插件弹窗里开启 `在 agentseller.temu.com 页面自动检查新品并同步` 后：

1. 保持 `https://agentseller.temu.com/` 页面打开。
2. 插件会主动请求商品接口第一页。
3. 第一次成功检查只记录当前第一页 SKC 作为基线。
4. 后续如果第一页出现新的 SKC，插件会自动抓取全部商品并同步到本项目。

这个自动检查仍然需要卖家中心页面处于打开状态，因为请求要复用页面里的登录态、`anti-content` 和 `mallid` 等风控上下文。

## 自动检查发货

在插件弹窗里开启 `在 seller.kuajingmaihuo.com 页面自动检查发货并同步` 后：

1. 保持 `https://seller.kuajingmaihuo.com/` 页面打开。
2. 插件会主动请求发货接口第一页，默认筛选 `status: 1`，每页 100 条。
3. 第一次成功检查只记录当前发货批次/发货单作为基线。
4. 后续如果出现新的发货批次或发货单，插件会自动抓取全部发货记录并同步到本项目。

这个自动检查同样需要卖家中心页面处于打开状态，用于复用当前登录态和风控上下文。

## 本地调试

1. 打开 Chrome 的 `chrome://extensions/`。
2. 开启“开发者模式”。
3. 点击“加载已解压的扩展程序”，选择本目录 `chrome-extension`。
4. 打开 Temu 卖家中心的商品或发货列表页，刷新或翻页触发页面接口。
5. 点击浏览器工具栏中的插件图标，登录本项目账号或粘贴 token，选择店铺后同步数据。

## 打包下载

运行项目根目录下的脚本：

```powershell
.\scripts\package-extension.ps1
```

如果部署到线上，打包时把系统 API 地址写进插件：

```powershell
.\scripts\package-extension.ps1 -ApiBase "https://your-domain.example/api"
```

也可以使用环境变量：

```powershell
$env:TEMU_TOOLS_EXTENSION_API_BASE = "https://your-domain.example/api"
.\scripts\package-extension.ps1
```

脚本会生成：

```text
frontend/public/downloads/temu-seller-sync-extension.zip
```

前端工具页会提供这个 zip 的下载入口。
