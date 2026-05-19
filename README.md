# terraform-cmdb

一个轻量的 Terraform state 机器资产展示工具。把 `terraform.tfstate` 或 Terraform JSON 放到 `states` 目录后，应用会自动解析常见云厂商的机器资源，并在页面中展示实例名称、Provider、规格、CPU、内存、磁盘、区域、内网 IP、公网 IP 和完整 attributes。

## 启动

```bash
go run .
```

打开 http://127.0.0.1:3000 查看资产。

默认会创建并读取当前目录下的 `states` 目录：

```bash
mkdir -p states
cp terraform.tfstate states/prod.tfstate
go run .
```

如果 Terraform state 在其他 Git 仓库里，可以把仓库或子目录软链接到 `states`：

```bash
mkdir -p states
ln -s /path/to/your/terraform-repo states/terraform-repo
go run .
```

扫描会递归跟随软链接目录，读取文件名以 `.json` 或 `.tfstate` 结尾的文件，并跳过 `.git`、`.terraform` 等目录。`terraform.tfstate.backup` 不会被解析。

页面上可以点击“刷新 states 目录”重新加载文件，也可以临时上传单个 state 文件调试。
临时上传接口限制为 100MB；更大的 state 建议放入 `states` 目录通过扫描读取。

## 页面功能

- 机器列表支持按机器名称搜索。
- 机器列表支持前端分页，默认每页 10 条，可切换为 20、50、100 条。
- “已加载文件”默认折叠，文件较多时不会占用太多页面空间。
- 每台机器可以展开查看完整 Terraform attributes。
- 内存列以 GB 展示，表头为 `内存(G)`，行内只显示数值。
- Swagger 中文接口文档地址：http://127.0.0.1:3000/swagger

## 接口

- `GET /`：实例列表页面
- `GET /swagger`：Swagger UI 中文接口文档
- `GET /swagger/openapi.json`：OpenAPI 3 JSON 文档
- `POST /reload`：重新扫描 `states` 目录
- `POST /upload`：临时上传表单字段名为 `state` 的 Terraform state JSON
- `GET /api/instances`：返回解析后的实例 JSON

## 发布

推送 `v*` tag 会触发 GitHub Actions 自动构建 Release 二进制包：

```bash
git tag v0.1.0
git push origin v0.1.0
```

Release assets 会包含：

- `terraform-cmdb_<version>_linux_amd64.tar.gz`
- `terraform-cmdb_<version>_linux_arm64.tar.gz`
- `terraform-cmdb_<version>_darwin_amd64.tar.gz`
- `terraform-cmdb_<version>_darwin_arm64.tar.gz`
- `checksums.txt`

## 当前识别范围

第一版优先识别常见机器资源，例如 `aws_instance`、`alicloud_instance`、`tencentcloud_instance`、`azurerm_linux_virtual_machine`、`google_compute_instance`、`vsphere_virtual_machine`、`openstack_compute_instance_v2` 等。

公网 IP 会额外从 EIP、公网 IP、floating IP 和 association 类资源里尝试关联回机器，例如通过 `instance_id`、`allocation_id`、`eip_id` 等字段把独立公网 IP 资源回填到实例。

CPU、内存、磁盘会从 Terraform state 里已有字段提取，例如 `cpu`、`cpu_core_count`、`memory`、`memory_gb`、`system_disk_size`、`root_block_device`、`data_disks` 等。如果 state 里只有实例规格名但没有展开资源配置，则暂时不会通过规格表反查 CPU 和内存。

页面详情里会展示完整 Terraform attributes，方便在不同云厂商字段不完全一致时直接查看原始数据。
