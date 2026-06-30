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
		rows.WriteString(renderMachineRow(machine))
	}

	if len(data.Machines) == 0 {
		emptyText := "还没有实例数据，上传 terraform.tfstate 或 state JSON 后查看。"
		if data.Static {
			emptyText = "还没有实例数据，请确认导出时 states 目录里有 Terraform state 文件。"
		}
		fmt.Fprintf(&rows, `<tr class="empty-row"><td colspan="10" class="px-6 py-20 text-center">
  <div class="mx-auto max-w-sm">
    <div class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-slate-100 text-slate-400">
      <svg class="h-7 w-7" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2"/></svg>
    </div>
    <p class="text-sm text-slate-500">%s</p>
  </div>
</td></tr>`, esc(emptyText))
	}
	noResultRow := `<tr id="no-results-row" class="hidden"><td colspan="10" class="px-6 py-16 text-center text-sm text-slate-500">没有匹配的机器，试试其他关键词。</td></tr>`

	errorBox := ""
	if data.LastError != "" {
		errorBox = fmt.Sprintf(`<div class="mb-6 flex items-start gap-3 rounded-2xl border border-red-200/80 bg-red-50 px-4 py-3 text-sm text-red-700 shadow-sm">
  <svg class="mt-0.5 h-5 w-5 shrink-0 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z"/></svg>
  <div>%s</div>
</div>`, esc(data.LastError))
	}

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

	apiLink := renderAPILink(data)
	actions := renderActions(data, stateDir)
	staticBanner := ""
	if data.Static {
		staticBanner = `<div class="mb-8 overflow-hidden rounded-2xl border border-orange-200/60 bg-gradient-to-r from-orange-50 via-white to-amber-50 p-5 shadow-sm">
  <div class="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
    <div class="flex items-start gap-4">
      <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-orange-500 to-amber-500 text-white shadow-lg shadow-orange-500/25">
        <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z"/></svg>
      </div>
      <div>
        <p class="text-xs font-semibold uppercase tracking-wider text-orange-600">Static Export</p>
        <h2 class="mt-1 text-lg font-semibold text-slate-900">静态资产报告</h2>
        <p class="mt-1 text-sm text-slate-600">由 <code class="rounded-md bg-white/80 px-1.5 py-0.5 text-xs text-slate-700 ring-1 ring-slate-200">go run . export</code> 生成，可部署到 Cloudflare Pages 等静态托管。</p>
      </div>
    </div>
    <a href="instances.json" class="inline-flex items-center justify-center gap-2 rounded-xl bg-slate-900 px-4 py-2.5 text-sm font-medium text-white shadow-sm transition hover:bg-slate-800">
      <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0 0L7.5 12m4.5 4.5V3"/></svg>
      下载 JSON
    </a>
  </div>
</div>`
	}

	return fmt.Sprintf(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Terraform CMDB</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
  <script src="https://cdn.tailwindcss.com"></script>
  <script>
    tailwind.config = {
      theme: {
        extend: {
          fontFamily: {
            sans: ['Inter', 'system-ui', 'sans-serif'],
            mono: ['JetBrains Mono', 'ui-monospace', 'monospace'],
          },
          colors: {
            brand: { 50: '#fff7ed', 100: '#ffedd5', 500: '#f97316', 600: '#ea580c', 700: '#c2410c' }
          },
          boxShadow: {
            card: '0 1px 2px rgba(15, 23, 42, 0.04), 0 8px 24px rgba(15, 23, 42, 0.06)',
          }
        }
      }
    }
  </script>
  <style>
    body { font-feature-settings: "cv02", "cv03", "cv04", "cv11"; }
    .mesh-bg {
      background-color: #f8fafc;
      background-image:
        radial-gradient(at 0%% 0%%, rgba(249, 115, 22, 0.08) 0px, transparent 50%%),
        radial-gradient(at 100%% 0%%, rgba(59, 130, 246, 0.06) 0px, transparent 50%%),
        radial-gradient(at 50%% 100%%, rgba(148, 163, 184, 0.08) 0px, transparent 50%%);
    }
    details[open] summary .chevron { transform: rotate(180deg); }
    .inventory-row { transition: background-color 0.15s ease; }
    ::-webkit-scrollbar { width: 6px; height: 6px; }
    ::-webkit-scrollbar-thumb { background: #cbd5e1; border-radius: 999px; }
  </style>
</head>
<body class="mesh-bg min-h-screen text-slate-900 antialiased">
  <div class="mx-auto max-w-[1400px] px-4 py-8 sm:px-6 lg:px-8">
    %s

    <header class="mb-8 flex flex-col gap-6 lg:flex-row lg:items-start lg:justify-between">
      <div class="max-w-2xl">
        <div class="inline-flex items-center gap-2 rounded-full border border-orange-200/70 bg-white/80 px-3 py-1 text-xs font-medium text-orange-700 shadow-sm backdrop-blur">
          <span class="h-1.5 w-1.5 rounded-full bg-orange-500"></span>
          Terraform CMDB
        </div>
        <h1 class="mt-4 text-3xl font-bold tracking-tight text-slate-950 sm:text-4xl">内网机器资产</h1>
        <p class="mt-3 text-sm leading-relaxed text-slate-600">%s</p>
      </div>
      %s
    </header>

    %s
    %s

    <section class="mb-8 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
      %s
      %s
      %s
      %s
    </section>

    <section class="overflow-hidden rounded-2xl border border-slate-200/80 bg-white/90 shadow-card backdrop-blur">
      <div class="flex flex-col gap-4 border-b border-slate-200/80 px-5 py-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h2 class="text-base font-semibold text-slate-900">实例列表</h2>
          <div id="table-summary" class="mt-1 text-xs text-slate-500"></div>
        </div>
        <div class="flex flex-col gap-2 sm:flex-row sm:items-center">
          <div class="relative">
            <svg class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z"/></svg>
            <input id="machine-search" class="w-full rounded-xl border border-slate-200 bg-slate-50/80 py-2.5 pl-10 pr-4 text-sm outline-none transition focus:border-orange-300 focus:bg-white focus:ring-2 focus:ring-orange-100 sm:w-72" type="search" placeholder="搜索机器名称…">
          </div>
          %s
        </div>
      </div>
      <div class="overflow-x-auto">
        <table class="min-w-full text-left text-sm">
          <thead class="bg-slate-50/90 text-[11px] font-semibold uppercase tracking-wider text-slate-500">
            <tr>
              <th class="px-5 py-3.5">名称</th>
              <th class="px-5 py-3.5">Provider</th>
              <th class="px-5 py-3.5">规格</th>
              <th class="px-5 py-3.5">CPU</th>
              <th class="px-5 py-3.5">内存(G)</th>
              <th class="px-5 py-3.5">磁盘</th>
              <th class="px-5 py-3.5">区域</th>
              <th class="px-5 py-3.5">内网 IP</th>
              <th class="px-5 py-3.5">公网 IP</th>
              <th class="px-5 py-3.5">详情</th>
            </tr>
          </thead>
          <tbody id="instances-body" class="divide-y divide-slate-100">%s%s</tbody>
        </table>
      </div>
      <div class="flex flex-col gap-3 border-t border-slate-200/80 bg-slate-50/50 px-5 py-4 text-sm text-slate-600 sm:flex-row sm:items-center sm:justify-between">
        <div class="flex items-center gap-2">
          <span class="text-slate-500">每页</span>
          <select id="page-size" class="rounded-lg border border-slate-200 bg-white px-2.5 py-1.5 text-sm outline-none focus:border-orange-300 focus:ring-2 focus:ring-orange-100">
            <option value="10" selected>10</option>
            <option value="20">20</option>
            <option value="50">50</option>
            <option value="100">100</option>
          </select>
          <span class="text-slate-500">条</span>
        </div>
        <div class="flex items-center gap-2">
          <button id="prev-page" class="rounded-lg border border-slate-200 bg-white px-3.5 py-1.5 font-medium text-slate-700 transition hover:border-slate-300 hover:bg-slate-50" type="button">上一页</button>
          <span id="page-info" class="min-w-[4.5rem] text-center text-xs font-medium text-slate-500"></span>
          <button id="next-page" class="rounded-lg border border-slate-200 bg-white px-3.5 py-1.5 font-medium text-slate-700 transition hover:border-slate-300 hover:bg-slate-50" type="button">下一页</button>
        </div>
      </div>
    </section>

    <footer class="mt-8 text-center text-xs text-slate-400">
      Terraform CMDB · 从 Terraform State 解析机器资产
    </footer>
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
        prevButton.classList.toggle("opacity-40", prevButton.disabled);
        prevButton.classList.toggle("cursor-not-allowed", prevButton.disabled);
        nextButton.classList.toggle("opacity-40", nextButton.disabled);
        nextButton.classList.toggle("cursor-not-allowed", nextButton.disabled);
        pageInfo.textContent = page + " / " + totalPages;
        tableSummary.textContent = query
          ? "匹配 " + matchedRows.length + " / 共 " + rows.length + " 台"
          : "共 " + rows.length + " 台";
      };

      searchInput.addEventListener("input", () => { page = 1; render(); });
      pageSizeSelect.addEventListener("change", () => { page = 1; render(); });
      prevButton.addEventListener("click", () => { page = Math.max(1, page - 1); render(); });
      nextButton.addEventListener("click", () => { page += 1; render(); });
      render();
    })();
  </script>
</body>
</html>`,
		staticBanner,
		renderIntro(data, stateDir),
		actions,
		errorBox,
		renderSourceFiles(data.SourceFiles),
		statCard("机器数量", fmt.Sprintf("%d", len(data.Machines)), "M", "from-orange-500 to-amber-500"),
		statCard("State 文件", esc(source), "S", "from-blue-500 to-cyan-500"),
		statCard("Terraform", esc(terraformVersion), "T", "from-violet-500 to-purple-500"),
		statCard("资源数", fmt.Sprintf("%d", data.RawResources), "R", "from-emerald-500 to-teal-500"),
		apiLink,
		rows.String(),
		noResultRow,
	)
}

func renderMachineRow(machine inventory.Machine) string {
	raw, _ := json.MarshalIndent(machine.Attributes, "", "  ")
	return fmt.Sprintf(`
<tr class="inventory-row group hover:bg-orange-50/40" data-machine-name="%s">
  <td class="px-5 py-4">
    <div class="flex items-center gap-3">
      <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-xs font-bold uppercase text-slate-500 group-hover:bg-white group-hover:shadow-sm">%s</div>
      <div class="min-w-0">
        <div class="truncate font-medium text-slate-900">%s</div>
        <div class="mt-0.5 truncate font-mono text-[11px] text-slate-400">%s</div>
      </div>
    </div>
  </td>
  <td class="px-5 py-4">%s</td>
  <td class="px-5 py-4"><span class="text-slate-700">%s</span></td>
  <td class="px-5 py-4"><span class="font-medium text-slate-800">%s</span></td>
  <td class="px-5 py-4"><span class="font-medium text-slate-800">%s</span></td>
  <td class="px-5 py-4 text-slate-600">%s</td>
  <td class="px-5 py-4 text-slate-600">%s</td>
  <td class="px-5 py-4">%s</td>
  <td class="px-5 py-4">%s</td>
  <td class="px-5 py-4">
    <details class="group/details">
      <summary class="flex cursor-pointer list-none items-center gap-1.5 text-xs font-medium text-orange-600 transition hover:text-orange-700">
        <span>Attributes</span>
        <svg class="chevron h-3.5 w-3.5 transition-transform" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M19.5 8.25l-7.5 7.5-7.5-7.5"/></svg>
      </summary>
      <pre class="mt-3 max-h-80 overflow-auto rounded-xl border border-slate-800 bg-slate-950 p-4 font-mono text-[11px] leading-relaxed text-slate-300 shadow-inner">%s</pre>
    </details>
  </td>
</tr>`,
		esc(strings.ToLower(machine.Name)),
		esc(machineInitial(machine.Name)),
		esc(machine.Name),
		esc(machine.ResourceAddress),
		badge(machine.Provider),
		esc(machine.InstanceType),
		esc(firstNonEmpty(machine.CPUCores, "-")),
		esc(firstNonEmpty(machine.Memory, "-")),
		renderDisks(machine.Disks),
		esc(firstNonEmpty(machine.Region, machine.Zone, "-")),
		renderIPs(machine.PrivateIPs, false),
		renderIPs(machine.PublicIPs, true),
		esc(string(raw)),
	)
}

func renderIntro(data IndexData, stateDir string) string {
	if data.Static {
		return fmt.Sprintf(`数据来自导出时扫描的 <code class="rounded-md bg-slate-100 px-1.5 py-0.5 text-xs font-medium text-slate-700 ring-1 ring-slate-200">%s</code> 目录，适合离线浏览与分享。`, esc(stateDir))
	}
	return fmt.Sprintf(`把 Terraform state 放到 <code class="rounded-md bg-slate-100 px-1.5 py-0.5 text-xs font-medium text-slate-700 ring-1 ring-slate-200">%s</code> 目录后刷新即可解析，也支持临时上传单个文件调试。`, esc(stateDir))
}

func renderActions(data IndexData, stateDir string) string {
	if data.Static {
		return `<div class="rounded-2xl border border-slate-200/80 bg-white/90 p-5 shadow-card backdrop-blur lg:min-w-80">
  <div class="flex items-center gap-3">
    <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-slate-100 text-slate-500">
      <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M2.036 12.322a1.012 1.012 0 010-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178z"/><path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/></svg>
    </div>
    <div>
      <div class="text-sm font-semibold text-slate-900">只读展示</div>
      <div class="mt-0.5 text-xs text-slate-500">无上传与刷新接口</div>
    </div>
  </div>
  <p class="mt-4 text-xs leading-relaxed text-slate-500">更新数据请本地执行 <code class="rounded bg-slate-100 px-1 py-0.5 text-[11px] text-slate-700">go run . export</code> 后重新部署 <code class="rounded bg-slate-100 px-1 py-0.5 text-[11px] text-slate-700">dist/</code>。</p>
</div>`
	}
	_ = stateDir
	return `<div class="flex flex-col gap-3 rounded-2xl border border-slate-200/80 bg-white/90 p-5 shadow-card backdrop-blur lg:min-w-80">
  <form action="/reload" method="post">
    <button class="flex w-full items-center justify-center gap-2 rounded-xl bg-gradient-to-r from-orange-500 to-amber-500 px-4 py-2.5 text-sm font-semibold text-white shadow-sm shadow-orange-500/20 transition hover:from-orange-600 hover:to-amber-600" type="submit">
      <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182"/></svg>
      刷新 states 目录
    </button>
  </form>
  <form action="/upload" method="post" enctype="multipart/form-data" class="flex flex-col gap-3">
    <label class="block w-full cursor-pointer rounded-xl border border-dashed border-slate-300 bg-slate-50/80 px-4 py-3 text-center text-xs text-slate-500 transition hover:border-orange-300 hover:bg-orange-50/50">
      <input class="sr-only" type="file" name="state" accept=".json,.tfstate,application/json" required onchange="this.form.querySelector('.file-label').textContent = this.files[0]?.name || '选择 .tfstate / .json 文件'">
      <span class="file-label">选择 .tfstate / .json 文件</span>
    </label>
    <button class="rounded-xl border border-slate-200 bg-white px-4 py-2.5 text-sm font-medium text-slate-700 transition hover:border-slate-300 hover:bg-slate-50" type="submit">临时上传单文件</button>
  </form>
</div>`
}

func renderAPILink(data IndexData) string {
	if data.Static {
		return `<a class="inline-flex items-center gap-1.5 rounded-xl border border-slate-200 bg-white px-3.5 py-2.5 text-xs font-medium text-slate-700 transition hover:border-slate-300 hover:bg-slate-50" href="instances.json">
  <svg class="h-3.5 w-3.5 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M17.25 6.75L22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3l-4.5 16.5"/></svg>
  JSON
</a>`
	}
	return `<a class="inline-flex items-center gap-1.5 rounded-xl border border-slate-200 bg-white px-3.5 py-2.5 text-xs font-medium text-slate-700 transition hover:border-slate-300 hover:bg-slate-50" href="/api/instances">
  <svg class="h-3.5 w-3.5 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M17.25 6.75L22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3l-4.5 16.5"/></svg>
  API JSON
</a>`
}

func statCard(label, value, icon, gradient string) string {
	return fmt.Sprintf(`<div class="group relative overflow-hidden rounded-2xl border border-slate-200/80 bg-white/90 p-5 shadow-card backdrop-blur transition hover:-translate-y-0.5 hover:shadow-lg">
  <div class="flex items-start justify-between">
    <div class="min-w-0 flex-1">
      <div class="text-xs font-medium text-slate-500">%s</div>
      <div class="mt-2 truncate text-2xl font-bold tracking-tight text-slate-900">%s</div>
    </div>
    <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br %s text-sm font-bold text-white shadow-sm">%s</div>
  </div>
</div>`, esc(label), value, gradient, esc(icon))
}

func renderDisks(disks []inventory.Disk) string {
	if len(disks) == 0 {
		return `<span class="text-slate-400">-</span>`
	}

	var out strings.Builder
	for i, disk := range disks {
		if i > 0 {
			out.WriteString(`<div class="mt-1">`)
		} else {
			out.WriteString(`<div>`)
		}
		label := disk.SizeGB
		if disk.Name != "" {
			label = disk.Name + " · " + label
		}
		fmt.Fprintf(&out, `<span class="whitespace-nowrap text-xs">%s</span>`, esc(label))
		if disk.Type != "" {
			fmt.Fprintf(&out, `<span class="ml-1 text-[10px] text-slate-400">%s</span>`, esc(disk.Type))
		}
		out.WriteString(`</div>`)
	}
	return out.String()
}

func renderIPs(ips []string, isPublic bool) string {
	if len(ips) == 0 {
		return `<span class="text-slate-400">-</span>`
	}
	cls := "bg-slate-100 text-slate-700 ring-slate-200/80"
	if isPublic {
		cls = "bg-emerald-50 text-emerald-700 ring-emerald-200/80"
	}
	var out strings.Builder
	for i, ip := range ips {
		if i > 0 {
			out.WriteString(" ")
		}
		fmt.Fprintf(&out, `<span class="inline-block rounded-md px-1.5 py-0.5 font-mono text-[11px] ring-1 %s">%s</span>`, cls, esc(ip))
	}
	return out.String()
}

func renderSourceFiles(files []string) string {
	if len(files) == 0 {
		return ""
	}

	var items strings.Builder
	for _, file := range files {
		fmt.Fprintf(&items, `<li class="truncate rounded-lg bg-slate-50 px-3 py-2 font-mono text-xs text-slate-600">%s</li>`, esc(file))
	}
	return fmt.Sprintf(`<details class="mb-6 overflow-hidden rounded-2xl border border-slate-200/80 bg-white/90 shadow-sm">
  <summary class="cursor-pointer px-5 py-4 text-sm font-medium text-slate-900 transition hover:bg-slate-50/80">
    已加载文件 <span class="ml-1 rounded-full bg-slate-100 px-2 py-0.5 text-xs font-semibold text-slate-600">%d</span>
  </summary>
  <ul class="grid gap-2 border-t border-slate-100 px-5 py-4 md:grid-cols-2">%s</ul>
</details>`, len(files), items.String())
}

func badge(value string) string {
	if value == "" {
		value = "unknown"
	}
	cls := providerBadgeClass(value)
	return fmt.Sprintf(`<span class="inline-flex items-center rounded-lg px-2.5 py-1 text-[11px] font-semibold uppercase tracking-wide %s">%s</span>`, cls, esc(value))
}

func providerBadgeClass(provider string) string {
	switch strings.ToLower(provider) {
	case "aws", "hashicorp/aws":
		return "bg-amber-100 text-amber-800 ring-1 ring-amber-200/80"
	case "alicloud", "aliyun":
		return "bg-orange-100 text-orange-800 ring-1 ring-orange-200/80"
	case "tencentcloud":
		return "bg-blue-100 text-blue-800 ring-1 ring-blue-200/80"
	case "google", "google_compute":
		return "bg-sky-100 text-sky-800 ring-1 ring-sky-200/80"
	case "azurerm", "azure":
		return "bg-indigo-100 text-indigo-800 ring-1 ring-indigo-200/80"
	case "vsphere":
		return "bg-violet-100 text-violet-800 ring-1 ring-violet-200/80"
	case "openstack":
		return "bg-rose-100 text-rose-800 ring-1 ring-rose-200/80"
	default:
		return "bg-slate-100 text-slate-700 ring-1 ring-slate-200/80"
	}
}

func machineInitial(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "?"
	}
	runes := []rune(name)
	if len(runes) == 0 {
		return "?"
	}
	return strings.ToUpper(string(runes[0]))
}

func esc(value string) string {
	return html.EscapeString(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
