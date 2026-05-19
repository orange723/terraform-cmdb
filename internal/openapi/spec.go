package openapi

func Spec() map[string]any {
	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "Terraform CMDB API",
			"description": "通过 Terraform state 文件解析并展示内网机器资产的接口文档。",
			"version":     "0.1.0",
		},
		"tags": []map[string]any{
			{
				"name":        "页面",
				"description": "资产页面和 Swagger 文档页面。",
			},
			{
				"name":        "资产",
				"description": "机器资产查询和 Terraform state 加载接口。",
			},
		},
		"paths": map[string]any{
			"/": map[string]any{
				"get": map[string]any{
					"tags":        []string{"页面"},
					"summary":     "资产列表页面",
					"description": "返回 Terraform CMDB 的 HTML 页面，页面包含机器列表、搜索、分页、state 刷新和临时上传入口。",
					"responses": map[string]any{
						"200": map[string]any{
							"description": "HTML 页面",
							"content": map[string]any{
								"text/html": map[string]any{
									"schema": map[string]any{
										"type": "string",
									},
								},
							},
						},
					},
				},
			},
			"/api/instances": map[string]any{
				"get": map[string]any{
					"tags":        []string{"资产"},
					"summary":     "获取机器资产列表",
					"description": "返回当前已加载 Terraform state 中解析出的机器资产。公网 IP 会从独立 EIP、public IP、floating IP 以及关联资源中尝试回填。",
					"responses": map[string]any{
						"200": map[string]any{
							"description": "机器资产列表",
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{
										"$ref": "#/components/schemas/InstanceListResponse",
									},
								},
							},
						},
					},
				},
			},
			"/reload": map[string]any{
				"post": map[string]any{
					"tags":        []string{"资产"},
					"summary":     "重新扫描 states 目录",
					"description": "重新递归扫描 states 目录，跟随软链接目录，读取文件名以 .json 或 .tfstate 结尾的 Terraform state 文件，并刷新内存中的资产列表。",
					"responses": map[string]any{
						"303": map[string]any{
							"description": "扫描完成后重定向回资产列表页面。",
						},
					},
				},
			},
			"/upload": map[string]any{
				"post": map[string]any{
					"tags":        []string{"资产"},
					"summary":     "临时上传单个 Terraform state 文件",
					"description": "上传一个 Terraform state JSON 文件并立即解析。该接口只替换当前内存中的资产数据，不会把文件写入 states 目录。",
					"requestBody": map[string]any{
						"required": true,
						"content": map[string]any{
							"multipart/form-data": map[string]any{
								"schema": map[string]any{
									"type":     "object",
									"required": []string{"state"},
									"properties": map[string]any{
										"state": map[string]any{
											"type":        "string",
											"format":      "binary",
											"description": "Terraform state JSON 文件，通常是 .tfstate 或 .json。",
										},
									},
								},
							},
						},
					},
					"responses": map[string]any{
						"303": map[string]any{
							"description": "解析成功后重定向回资产列表页面。",
						},
						"400": map[string]any{
							"description": "没有选择文件、文件读取失败或 Terraform state JSON 解析失败。",
							"content": map[string]any{
								"text/plain": map[string]any{
									"schema": map[string]any{
										"type": "string",
									},
								},
							},
						},
					},
				},
			},
			"/swagger/openapi.json": map[string]any{
				"get": map[string]any{
					"tags":        []string{"页面"},
					"summary":     "获取 OpenAPI JSON",
					"description": "返回本服务的 OpenAPI 3 JSON 文档，Swagger UI 会读取该接口渲染接口文档。",
					"responses": map[string]any{
						"200": map[string]any{
							"description": "OpenAPI JSON 文档",
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{
										"type": "object",
									},
								},
							},
						},
					},
				},
			},
		},
		"components": map[string]any{
			"schemas": map[string]any{
				"InstanceListResponse": map[string]any{
					"type":        "object",
					"description": "机器资产列表响应。",
					"properties": map[string]any{
						"file_name": map[string]any{
							"type":        "string",
							"description": "当前数据来源。目录扫描时会显示 states 目录和文件数量，临时上传时会显示上传文件名。",
							"example":     "states (2 files)",
						},
						"terraform": map[string]any{
							"type":        "string",
							"description": "已加载 state 文件中的 Terraform 版本，多个版本会用逗号拼接。",
							"example":     "1.14.7",
						},
						"raw_resources": map[string]any{
							"type":        "integer",
							"description": "已读取的 Terraform resource 数量。",
							"example":     12,
						},
						"count": map[string]any{
							"type":        "integer",
							"description": "解析出的机器数量。",
							"example":     2,
						},
						"source_files": map[string]any{
							"type":        "array",
							"description": "本次目录扫描实际加载的 state 文件路径。",
							"items": map[string]any{
								"type": "string",
							},
						},
						"instances": map[string]any{
							"type":        "array",
							"description": "机器资产列表。",
							"items": map[string]any{
								"$ref": "#/components/schemas/Machine",
							},
						},
					},
				},
				"Machine": map[string]any{
					"type":        "object",
					"description": "从 Terraform state 归一化后的机器资产。",
					"properties": map[string]any{
						"id": map[string]any{
							"type":        "string",
							"description": "云厂商实例 ID。",
							"example":     "i-2zec37hlpvpavqzcv7ga",
						},
						"name": map[string]any{
							"type":        "string",
							"description": "机器名称，优先使用实例名称或 Name 标签。",
							"example":     "ecs-devops-prod-beijing-harbor-00",
						},
						"provider": map[string]any{
							"type":        "string",
							"description": "Terraform Provider 名称。",
							"example":     "alicloud",
						},
						"resource_type": map[string]any{
							"type":        "string",
							"description": "Terraform 资源类型。",
							"example":     "alicloud_instance",
						},
						"resource_name": map[string]any{
							"type":        "string",
							"description": "Terraform 资源名。",
							"example":     "ecs-devops-prod-beijing-harbor-00",
						},
						"resource_address": map[string]any{
							"type":        "string",
							"description": "Terraform 资源地址。",
							"example":     "alicloud_instance.ecs-devops-prod-beijing-harbor-00",
						},
						"region": map[string]any{
							"type":        "string",
							"description": "区域。如果 state 中没有该字段，可能为空。",
							"example":     "cn-beijing",
						},
						"zone": map[string]any{
							"type":        "string",
							"description": "可用区。",
							"example":     "cn-beijing-l",
						},
						"status": map[string]any{
							"type":        "string",
							"description": "实例状态。",
							"example":     "Running",
						},
						"instance_type": map[string]any{
							"type":        "string",
							"description": "云厂商实例规格。",
							"example":     "ecs.c6.large",
						},
						"cpu_cores": map[string]any{
							"type":        "string",
							"description": "CPU 核数，直接来自 Terraform state 中的 CPU 或 vCPU 字段。",
							"example":     "2",
						},
						"memory": map[string]any{
							"type":        "string",
							"description": "内存大小，单位为 GB，响应值不带单位。",
							"example":     "4",
						},
						"disks": map[string]any{
							"type":        "array",
							"description": "磁盘列表，包含系统盘和数据盘。",
							"items": map[string]any{
								"$ref": "#/components/schemas/Disk",
							},
						},
						"private_ips": map[string]any{
							"type":        "array",
							"description": "内网 IP 列表。",
							"items": map[string]any{
								"type": "string",
							},
							"example": []string{"172.18.50.110"},
						},
						"public_ips": map[string]any{
							"type":        "array",
							"description": "公网 IP 列表，包含从 EIP 关联资源回填的地址。",
							"items": map[string]any{
								"type": "string",
							},
							"example": []string{"47.93.195.26"},
						},
						"tags": map[string]any{
							"type":                 "object",
							"description":          "Terraform state 中的 tags、labels 或 metadata。",
							"additionalProperties": true,
						},
						"attributes": map[string]any{
							"type":                 "object",
							"description":          "Terraform state 中该机器资源的完整 attributes 原始内容。",
							"additionalProperties": true,
						},
					},
				},
				"Disk": map[string]any{
					"type":        "object",
					"description": "机器磁盘信息。",
					"properties": map[string]any{
						"name": map[string]any{
							"type":        "string",
							"description": "磁盘名称或来源字段名。",
							"example":     "system",
						},
						"type": map[string]any{
							"type":        "string",
							"description": "磁盘类型。",
							"example":     "cloud_essd",
						},
						"size_gb": map[string]any{
							"type":        "string",
							"description": "磁盘大小，通常以 GB 展示。",
							"example":     "100 GB",
						},
					},
				},
			},
		},
	}
}
