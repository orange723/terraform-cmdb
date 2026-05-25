package web

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"

	"terraform-cmdb/internal/inventory"
)

type IndexData struct {
	FileName     string
	SourceFiles  []string
	StateDir     string
	Terraform    string
	Machines     []inventory.Machine
	LastError    string
	RawResources int
	Static       bool
}

func RenderIndex(data IndexData) string {
	var rows strings.Builder
	for _, machine := range data.Machines {
		raw, _ := json.MarshalIndent(machine.Attributes, "", "  ")
		fmt.Fprintf(&rows, `
<tr class="inventory-row border-b border-slate-200/70 hover:bg-slate-50" data-machine-name="%s">
  <td class="px-4 py-3">
    <div class="font-medium text-slate-950">%s</div>
    <div class="text-xs text-slate-500">%s</div>
  </td>
  <td class="px-4 py-3">%s</td>
  <td class="px-4 py-3">%s</td>
  <td class="px-4 py-3">%s</td>
  <td class="px-4 py-3">%s</td>
  <td class="px-4 py-3">%s</td>
  <td class="px-4 py-3">%s</td>
  <td class="px-4 py-3">%s</td>
  <td class="px-4 py-3">%s</td>
  <td class="px-4 py-3">
    <details>
      <summary class="cursor-pointer text-blue-600">查看 attributes</summary>
      <pre class="mt-3 max-h-96 overflow-auto rounded-lg bg-slate-950 p-3 text-xs text-slate-100">%s</pre>
    </details>
  </td>
</tr>`,
			esc(strings.ToLower(machine.Name)),
			esc(machine.Name),
			esc(machine.ResourceAddress),
			badge(machine.Provider),
			esc(machine.InstanceType),
			esc(firstNonEmpty(machine.CPUCores, "-")),
			esc(firstNonEmpty(machine.Memory, "-")),
			renderDisks(machine.Disks),
			esc(firstNonEmpty(machine.Region, machine.Zone, "-")),
			esc(joinOrDash(machine.PrivateIPs)),
			esc(joinOrDash(machine.PublicIPs)),
			esc(string(raw)),
		)
	}

	if len(data.Machines) == 0 {
		emptyText := "还没有实例数据，上传 terraform.tfstate 或 state JSON 后查看。"
		if data.Static {
			emptyText = "还没有实例数据，请确认导出时 states 目录里有 Terraform state 文件。"
		}
		fmt.Fprintf(&rows, `<tr class="empty-row"><td colspan="10" class="px-4 py-16 text-center text-slate-500">%s</td></tr>`, esc(emptyText))
	}
	noResultRow := `<tr id="no-results-row" class="hidden"><td colspan="10" class="px-4 py-16 text-center text-slate-500">没有匹配的机器。</td></tr>`

	errorBox := ""
	if data.LastError != "" {
		errorBox = fmt.Sprintf(`<div class="mb-6 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">%s</div>`, esc(data.LastError))
	}
	sourceFiles := renderSourceFiles(data.SourceFiles)

	source := "尚未上传"
	if data.FileName != "" {
		source = data.FileName
	}
	stateDir := data.StateDir
	if stateDir == "" {
		stateDir = "states"
	}
	terraformVersion := data.Terraform
	if terraformVersion == "" {
		terraformVersion = "-"
	}
	intro := fmt.Sprintf(`把 Terraform state JSON 放到 <code class="rounded bg-slate-200 px-1">%s</code>，刷新后自动解析；也可以临时上传单个文件。`, esc(stateDir))
	actions := `<div class="flex flex-col gap-3 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm md:min-w-96">
        <form action="/reload" method="post">
          <button class="w-full rounded-lg bg-orange-500 px-4 py-2 text-sm font-semibold text-white hover:bg-orange-600" type="submit">刷新 states 目录</button>
        </form>
        <form action="/upload" method="post" enctype="multipart/form-data" class="flex flex-col gap-3">
          <input class="block w-full rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-sm" type="file" name="state" accept=".json,.tfstate,application/json" required>
          <button class="rounded-lg border border-slate-200 px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50" type="submit">临时上传单文件</button>
        </form>
      </div>`
	apiLink := `<a class="text-sm font-medium text-blue-600 hover:text-blue-700" href="/api/instances">API JSON</a>`
	if data.Static {
		intro = fmt.Sprintf(`静态导出页面，数据来自导出时扫描的 <code class="rounded bg-slate-200 px-1">%s</code>。页面不包含上传、刷新或服务端接口。`, esc(stateDir))
		actions = `<div class="rounded-2xl border border-slate-200 bg-white p-4 text-sm text-slate-600 shadow-sm md:min-w-96">
        <div class="font-medium text-slate-900">静态展示模式</div>
        <div class="mt-1">重新生成数据请在本地运行 <code class="rounded bg-slate-100 px-1">go run . export</code> 后重新部署。</div>
      </div>`
		apiLink = `<a class="text-sm font-medium text-blue-600 hover:text-blue-700" href="instances.json">JSON 数据</a>`
	}

	return fmt.Sprintf(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Terraform CMDB</title>
  <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="min-h-screen bg-slate-100 text-slate-900">
  <div class="mx-auto max-w-7xl px-6 py-8">
    <header class="mb-8 flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
      <div>
        <p class="text-sm font-semibold uppercase tracking-wide text-orange-500">Terraform CMDB</p>
        <h1 class="mt-2 text-3xl font-semibold tracking-tight">内网机器资产</h1>
        <p class="mt-2 text-sm text-slate-500">%s</p>
      </div>
      %s
    </header>

    %s
    %s

    <section class="mb-6 grid gap-4 md:grid-cols-4">
      <div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
        <div class="text-sm text-slate-500">机器数量</div>
        <div class="mt-2 text-3xl font-semibold">%d</div>
      </div>
      <div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
        <div class="text-sm text-slate-500">State 文件</div>
        <div class="mt-2 truncate text-lg font-semibold">%s</div>
      </div>
      <div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
        <div class="text-sm text-slate-500">Terraform</div>
        <div class="mt-2 text-lg font-semibold">%s</div>
      </div>
      <div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
        <div class="text-sm text-slate-500">资源数</div>
        <div class="mt-2 text-3xl font-semibold">%d</div>
      </div>
    </section>

    <section class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
      <div class="flex flex-col gap-3 border-b border-slate-200 px-4 py-3 md:flex-row md:items-center md:justify-between">
        <div>
          <h2 class="font-semibold">实例列表</h2>
          <div id="table-summary" class="mt-1 text-xs text-slate-500"></div>
        </div>
        <div class="flex flex-col gap-2 md:flex-row md:items-center">
          <input id="machine-search" class="w-full rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-sm outline-none focus:border-orange-400 md:w-72" type="search" placeholder="按机器名称搜索">
          %s
        </div>
      </div>
      <div class="overflow-x-auto">
        <table class="min-w-full text-left text-sm">
          <thead class="bg-slate-50 text-xs uppercase tracking-wide text-slate-500">
            <tr>
              <th class="px-4 py-3">名称</th>
              <th class="px-4 py-3">Provider</th>
              <th class="px-4 py-3">规格</th>
              <th class="px-4 py-3">CPU</th>
              <th class="px-4 py-3">内存(G)</th>
              <th class="px-4 py-3">磁盘</th>
              <th class="px-4 py-3">区域</th>
              <th class="px-4 py-3">内网 IP</th>
              <th class="px-4 py-3">公网 IP</th>
              <th class="px-4 py-3">详情</th>
            </tr>
          </thead>
          <tbody id="instances-body">%s%s</tbody>
        </table>
      </div>
      <div class="flex flex-col gap-3 border-t border-slate-200 px-4 py-3 text-sm text-slate-600 md:flex-row md:items-center md:justify-between">
        <div class="flex items-center gap-2">
          <span>每页</span>
          <select id="page-size" class="rounded-lg border border-slate-200 bg-white px-2 py-1">
            <option value="10" selected>10</option>
            <option value="20">20</option>
            <option value="50">50</option>
            <option value="100">100</option>
          </select>
          <span>条</span>
        </div>
        <div class="flex items-center gap-3">
          <button id="prev-page" class="rounded-lg border border-slate-200 px-3 py-1 font-medium text-slate-700 hover:bg-slate-50" type="button">上一页</button>
          <span id="page-info"></span>
          <button id="next-page" class="rounded-lg border border-slate-200 px-3 py-1 font-medium text-slate-700 hover:bg-slate-50" type="button">下一页</button>
        </div>
      </div>
    </section>
  </div>
  <script>
    (() => {
      const rows = Array.from(document.querySelectorAll(".inventory-row"));
      const emptyRow = document.querySelector(".empty-row");
      const noResultsRow = document.getElementById("no-results-row");
      const searchInput = document.getElementById("machine-search");
      const pageSizeSelect = document.getElementById("page-size");
      const prevButton = document.getElementById("prev-page");
      const nextButton = document.getElementById("next-page");
      const pageInfo = document.getElementById("page-info");
      const tableSummary = document.getElementById("table-summary");
      let page = 1;

      const render = () => {
        const query = (searchInput.value || "").trim().toLowerCase();
        const pageSize = Number(pageSizeSelect.value);
        const matchedRows = rows.filter((row) => row.dataset.machineName.includes(query));
        const totalPages = Math.max(1, Math.ceil(matchedRows.length / pageSize));
        page = Math.min(page, totalPages);

        rows.forEach((row) => row.classList.add("hidden"));
        const start = (page - 1) * pageSize;
        matchedRows.slice(start, start + pageSize).forEach((row) => row.classList.remove("hidden"));

        if (emptyRow) {
          emptyRow.classList.toggle("hidden", rows.length > 0);
        }
        if (noResultsRow) {
          noResultsRow.classList.toggle("hidden", rows.length === 0 || matchedRows.length > 0);
        }

        prevButton.disabled = page <= 1;
        nextButton.disabled = page >= totalPages;
        prevButton.classList.toggle("opacity-50", prevButton.disabled);
        nextButton.classList.toggle("opacity-50", nextButton.disabled);
        pageInfo.textContent = page + " / " + totalPages;
        tableSummary.textContent = query
          ? "匹配 " + matchedRows.length + " / 共 " + rows.length + " 台"
          : "共 " + rows.length + " 台";
      };

      searchInput.addEventListener("input", () => {
        page = 1;
        render();
      });
      pageSizeSelect.addEventListener("change", () => {
        page = 1;
        render();
      });
      prevButton.addEventListener("click", () => {
        page = Math.max(1, page - 1);
        render();
      });
      nextButton.addEventListener("click", () => {
        page += 1;
        render();
      });
      render();
    })();
  </script>
</body>
</html>`,
		intro,
		actions,
		errorBox,
		sourceFiles,
		len(data.Machines),
		esc(source),
		esc(terraformVersion),
		data.RawResources,
		apiLink,
		rows.String(),
		noResultRow,
	)
}

func renderDisks(disks []inventory.Disk) string {
	if len(disks) == 0 {
		return "-"
	}

	var out strings.Builder
	for _, disk := range disks {
		label := disk.SizeGB
		if disk.Name != "" {
			label = disk.Name + ": " + label
		}
		if disk.Type != "" {
			label += " (" + disk.Type + ")"
		}
		fmt.Fprintf(&out, `<div class="whitespace-nowrap">%s</div>`, esc(label))
	}
	return out.String()
}

func renderSourceFiles(files []string) string {
	if len(files) == 0 {
		return ""
	}

	var items strings.Builder
	for _, file := range files {
		fmt.Fprintf(&items, `<li class="truncate">%s</li>`, esc(file))
	}
	return fmt.Sprintf(`<details class="mb-6 rounded-xl border border-slate-200 bg-white px-4 py-3 text-sm text-slate-600">
  <summary class="cursor-pointer font-medium text-slate-900">已加载文件（%d）</summary>
  <ul class="mt-3 grid gap-1 md:grid-cols-2">%s</ul>
</details>`, len(files), items.String())
}

func badge(value string) string {
	if value == "" {
		value = "unknown"
	}
	return fmt.Sprintf(`<span class="inline-flex rounded-full bg-slate-100 px-2.5 py-1 text-xs font-medium text-slate-700">%s</span>`, esc(value))
}

func esc(value string) string {
	return html.EscapeString(value)
}

func joinOrDash(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ", ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
