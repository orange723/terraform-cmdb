# terraform-cmdb

一个轻量的 Terraform state 机器资产展示工具。把 `terraform.tfstate` 或 Terraform JSON 放到 `states` 目录后，应用会自动解析机器资源，并展示名称、Provider、规格、CPU、内存、磁盘、区域、内网 IP、公网 IP 和完整 attributes。

## 快速启动

```bash
# 下载 Release 里的压缩包后解压
tar -xzf terraform-cmdb_<version>_<os>_<arch>.tar.gz

# 启动服务
./terraform-cmdb
```

打开 http://127.0.0.1:3000 查看资产。

程序启动时会自动创建并读取当前目录下的 `states` 目录。

## 准备 State 文件

直接复制 state 文件：

```bash
mkdir -p states
cp terraform.tfstate states/prod.tfstate
./terraform-cmdb
```

如果 Terraform state 在其他 Git 仓库里，可以把仓库或子目录软链接到 `states`：

```bash
mkdir -p states
ln -s /path/to/your/terraform-repo states/terraform-repo
./terraform-cmdb
```

扫描规则：

- 递归扫描 `states` 目录。
- 支持软链接目录。
- 读取文件名以 `.json` 或 `.tfstate` 结尾的文件。
- 跳过 `.git`、`.terraform`、`node_modules` 等目录。
- 不解析 `terraform.tfstate.backup`。

页面上可以点击“刷新 states 目录”重新加载文件，也可以临时上传单个 state 文件调试。临时上传限制为 100MB；更大的 state 建议放入 `states` 目录。

## 页面功能

- 按机器名称搜索。
- 前端分页，默认每页 10 条，可切换为 20、50、100 条。
- 已加载文件默认折叠展示。
- 每台机器可展开查看完整 Terraform attributes。
- 公网 IP 会从 EIP、public IP、floating IP 和 association 类资源中尝试回填。
- CPU、内存、磁盘会从 state 中已有字段提取。
- vSphere 机器会额外从 `default_ip_address`、`guest_ip_addresses` 以及 `guestinfo.metadata` / `guestinfo.userdata` 的 cloud-init 网络配置中提取内网 IP。
- vSphere 机器的“区域”列会显示运行宿主机；如果 state 里有 `data.vsphere_host`，会优先显示宿主机名，否则显示 `host_system_id`。
- Swagger 中文接口文档：http://127.0.0.1:3000/swagger

## 接口

- `GET /`：实例列表页面
- `GET /swagger`：Swagger UI 中文接口文档
- `GET /swagger/openapi.json`：OpenAPI 3 JSON 文档
- `POST /reload`：重新扫描 `states` 目录
- `POST /upload`：临时上传表单字段名为 `state` 的 Terraform state JSON
- `GET /api/instances`：返回解析后的实例 JSON

## 源码运行

需要 Go 环境：

```bash
go run .
```

测试：

```bash
go test ./...
```

## Release 构建

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

优先识别常见机器资源，例如 `aws_instance`、`alicloud_instance`、`tencentcloud_instance`、`azurerm_linux_virtual_machine`、`google_compute_instance`、`vsphere_virtual_machine`、`openstack_compute_instance_v2` 等。

CPU、内存、磁盘只从 Terraform state 已有字段提取，例如 `cpu`、`cpu_core_count`、`memory`、`memory_gb`、`system_disk_size`、`root_block_device`、`data_disks` 等。如果 state 里只有实例规格名但没有展开资源配置，暂时不会通过云厂商规格表反查。

vSphere 场景会做少量特殊处理：支持解码 base64 或 gzip+base64 的 `guestinfo.metadata` / `guestinfo.userdata`，从 cloud-init 网络 YAML 中提取静态内网 IP；同时会把 `host_system_id` 或关联的 `vsphere_host` data source 名称展示到区域列，便于查看机器所在 ESXi 主机。
