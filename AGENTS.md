# terraform-cmdb

## 描述

通过读取 Terraform state 文件展示当前机器资产，不限制云平台。项目使用 Go + Fiber v3，暂时不使用数据库，解析结果保存在内存中，前端由 Fiber 直接返回 HTML，样式走 Tailwind CDN，整体偏 Cloudflare 控制台风格。

## 当前实现

- 默认读取当前目录下的 `states` 目录。
- `states` 支持放 `.json`、`.tfstate` 文件，也支持软链接到其他 Terraform Git 仓库或子目录。
- 扫描会递归跟随软链接目录，跳过 `.git`、`.terraform`、`node_modules`、`.idea`、`.vscode` 等目录。
- 只读取文件名以 `.json` 或 `.tfstate` 结尾的文件，`terraform.tfstate.backup` 不解析。
- 页面支持刷新 `states` 目录，也保留临时上传单个 state 文件调试。
- Fiber `BodyLimit` 设置为 100MB，临时上传超过该大小时建议改用 `states` 目录扫描。
- 页面支持按机器名称搜索和前端分页，默认每页 10 条。
- “已加载文件”和单机 attributes 都默认折叠展示。
- 已提供 Swagger 中文接口文档：`GET /swagger`，OpenAPI JSON：`GET /swagger/openapi.json`。
- 支持 `go run . export` 静态导出到 `dist/index.html` 和 `dist/instances.json`，用于 Cloudflare Pages 等纯静态托管；静态页面不包含上传、刷新或后端接口。

## 资产解析约定

- 优先识别常见机器资源，例如 `aws_instance`、`alicloud_instance`、`tencentcloud_instance`、`azurerm_linux_virtual_machine`、`google_compute_instance`、`vsphere_virtual_machine`、`openstack_compute_instance_v2` 等。
- 公网 IP 需要从独立公网 IP 资源和 association 资源回填，例如通过 `instance_id`、`allocation_id`、`eip_id` 等字段关联机器。
- vSphere 机器会额外从 `default_ip_address`、`guest_ip_addresses`、`guestinfo.metadata`、`guestinfo.userdata` 中提取私网 IP；`guestinfo.*` 支持 base64 和 gzip+base64 的 cloud-init 网络配置。
- vSphere 机器的区域字段用于展示所在宿主机，优先通过 `data.vsphere_host` 把 `host_system_id` 转成主机名，否则显示 `host_system_id`。
- CPU、内存、磁盘只从 state 已有字段提取；如果 state 里只有实例规格名，暂时不通过云厂商规格表或 API 反查。
- 内存统一按 GB 展示，表头为 `内存(G)`，行内只显示数值。
- 详情里保留完整 Terraform attributes，方便排查不同云厂商字段差异。

## 代码结构

- `main.go`：启动入口，创建 `states` 目录，初始化 store 和 server。
- `internal/server`：Fiber app、路由和 handler。
- `internal/inventory`：内存 store 和资产模型。
- `internal/statefiles`：state 目录扫描和软链接递归加载。
- `internal/terraformstate`：Terraform state 解析、机器资源归一化、公网 IP 关联。
- `internal/web`：HTML 页面渲染。
- `internal/openapi`：OpenAPI 3 文档定义。

## 发布构建

- GitHub Actions workflow 位于 `.github/workflows/release.yml`。
- 推送 `v*` tag 会自动运行测试，并构建 Linux/macOS 的 amd64、arm64 二进制包。
- Release assets 包含四个平台包和 `checksums.txt`。
- 本地二进制、`dist/` 和 `states/` 已通过 `.gitignore` 忽略。
