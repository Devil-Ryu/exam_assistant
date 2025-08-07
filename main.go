package main

import (
	"embed"
	_ "embed"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

// main function serves as the application's entry point. It initializes the application, creates a window,
// and starts a goroutine that emits a time-based event every second. It subsequently runs the application and
// logs any error that might occur.
func main() {

	// Create a new Wails application by providing the necessary options.
	// Variables 'Name' and 'Description' are for application metadata.
	// 'Assets' configures the asset server with the 'FS' variable pointing to the frontend files.
	// 'Bind' is a list of Go struct instances. The frontend has access to the methods of these instances.
	// 'Mac' options tailor the application when running an macOS.
	app := application.New(application.Options{
		Name:        "考试小助手",
		Description: "一个帮助用户快速查找答案的考试助手工具",
		Services: []application.Service{
			application.NewService(&ExamService{}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// Create a new window with the necessary options.
	// 'Title' is the title of the window.
	// 'Mac' options tailor the window when running on macOS.
	// 'BackgroundColour' is the background colour of the window.
	// 'URL' is the URL that will be loaded into the webview.
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "考试小助手",
		Width:  1200,
		Height: 840,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(255, 255, 255),
		URL:              "/",
	})

	// 启动HTTP服务器
	go startHTTPServer()

	// Run the application. This blocks until the application has been exited.
	err := app.Run()

	// If an error occurred while running the application, log it and exit.
	if err != nil {
		log.Fatal(err)
	}
}

// startHTTPServer 启动HTTP服务器
func startHTTPServer() {
	// 创建HTTP服务器
	mux := http.NewServeMux()

	// 注册搜索接口
	mux.HandleFunc("/api/search", handleSearch)

	// 注册CSV解析接口
	mux.HandleFunc("/api/parse-csv", handleParseCSV)

	// 注册设置全局答案接口
	mux.HandleFunc("/api/set-global-answers", handleSetGlobalAnswers)

	// 注册获取全局答案接口
	mux.HandleFunc("/api/get-global-answers", handleGetGlobalAnswers)

	// 注册OCR测试接口
	mux.HandleFunc("/api/test-ocr", handleTestOCR)

	// 注册截图接口
	mux.HandleFunc("/api/take-screenshot", handleTakeScreenshot)

	// 注册执行OCR接口
	mux.HandleFunc("/api/perform-ocr", handlePerformOCR)

	// 启动服务器
	port := ":8088"
	log.Printf("HTTP服务器启动在端口 %s", port)

	// 优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := http.ListenAndServe(port, mux); err != nil {
			log.Printf("HTTP服务器错误: %v", err)
		}
	}()

	<-sigChan
	log.Println("正在关闭HTTP服务器...")
}
