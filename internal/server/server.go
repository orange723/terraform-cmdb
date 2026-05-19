package server

import (
	"io"

	"github.com/gofiber/fiber/v3"

	"terraform-cmdb/internal/inventory"
	"terraform-cmdb/internal/openapi"
	"terraform-cmdb/internal/statefiles"
	"terraform-cmdb/internal/terraformstate"
	"terraform-cmdb/internal/web"
)

type Config struct {
	AppName  string
	StateDir string
}

type Server struct {
	app      *fiber.App
	store    *inventory.Store
	stateDir string
}

func New(store *inventory.Store, config Config) *Server {
	if config.AppName == "" {
		config.AppName = "terraform-cmdb"
	}
	if config.StateDir == "" {
		config.StateDir = "states"
	}

	server := &Server{
		app: fiber.New(fiber.Config{
			AppName:      config.AppName,
			ServerHeader: config.AppName,
			BodyLimit:    100 * 1024 * 1024,
		}),
		store:    store,
		stateDir: config.StateDir,
	}
	server.registerRoutes()
	return server
}

func (s *Server) App() *fiber.App {
	return s.app
}

func (s *Server) registerRoutes() {
	s.app.Get("/", s.handleIndex)
	s.app.Get("/swagger", s.handleSwaggerUI)
	s.app.Get("/swagger/openapi.json", s.handleOpenAPI)
	s.app.Get("/api/instances", s.handleInstances)
	s.app.Post("/upload", s.handleUpload)
	s.app.Post("/reload", s.handleReload)
}

// handleIndex 返回资产列表 HTML 页面，包含搜索、分页、刷新和临时上传入口。
func (s *Server) handleIndex(c fiber.Ctx) error {
	snapshot := s.store.Snapshot()

	c.Type("html", "utf-8")
	return c.SendString(web.RenderIndex(web.IndexData{
		FileName:     snapshot.FileName,
		Terraform:    snapshot.Terraform,
		Machines:     snapshot.Machines,
		LastError:    snapshot.LastError,
		RawResources: snapshot.RawResources,
		SourceFiles:  snapshot.SourceFiles,
		StateDir:     s.stateDir,
	}))
}

// handleSwaggerUI 返回 Swagger UI 页面，用于查看中文接口文档。
func (s *Server) handleSwaggerUI(c fiber.Ctx) error {
	c.Type("html", "utf-8")
	return c.SendString(swaggerHTML())
}

// handleOpenAPI 返回 OpenAPI 3 JSON 文档。
func (s *Server) handleOpenAPI(c fiber.Ctx) error {
	return c.JSON(openapi.Spec())
}

// handleInstances 返回当前内存中的机器资产 JSON。
func (s *Server) handleInstances(c fiber.Ctx) error {
	snapshot := s.store.Snapshot()

	return c.JSON(fiber.Map{
		"file_name":     snapshot.FileName,
		"terraform":     snapshot.Terraform,
		"raw_resources": snapshot.RawResources,
		"count":         len(snapshot.Machines),
		"source_files":  snapshot.SourceFiles,
		"instances":     snapshot.Machines,
	})
}

// handleUpload 接收单个 Terraform state 文件并临时替换当前资产数据。
func (s *Server) handleUpload(c fiber.Ctx) error {
	file, err := c.FormFile("state")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("请选择 Terraform state JSON 文件")
	}

	opened, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("读取上传文件失败")
	}
	defer opened.Close()

	content, err := io.ReadAll(opened)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("读取上传文件失败")
	}

	result, err := terraformstate.Parse(content)
	s.store.Replace(inventory.Snapshot{
		FileName:     file.Filename,
		Terraform:    result.Terraform,
		RawResources: result.RawResources,
		Machines:     result.Machines,
		LastError:    errorString(err),
	})
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	return c.Redirect().To("/")
}

// handleReload 重新扫描 states 目录并刷新当前资产数据。
func (s *Server) handleReload(c fiber.Ctx) error {
	s.LoadStateDirectory()
	return c.Redirect().To("/")
}

func (s *Server) LoadStateDirectory() {
	result := statefiles.LoadDirectory(s.stateDir)
	s.store.Replace(result.Snapshot)
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func swaggerHTML() string {
	return `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Terraform CMDB API 文档</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = () => {
      SwaggerUIBundle({
        url: "/swagger/openapi.json",
        dom_id: "#swagger-ui",
        deepLinking: true,
        defaultModelsExpandDepth: 1,
        defaultModelExpandDepth: 2
      });
    };
  </script>
</body>
</html>`
}
