package main

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// ExamService 考试助手服务
type ExamService struct{}

// OCRConfig OCR配置
type OCRConfig struct {
	Mode   string `json:"mode"`   // "online" 或 "local"
	URL    string `json:"url"`    // 在线OCR URL
	APIKey string `json:"apiKey"` // API密钥
	Status string `json:"status"` // 连接状态
}

// ImportConfig 导入配置
type ImportConfig struct {
	FileType  string `json:"fileType"`  // "excel" 或 "csv"
	Encoding  string `json:"encoding"`  // 文件编码
	Delimiter string `json:"delimiter"` // 答案分隔符
}

// ScreenshotArea 截图区域
type ScreenshotArea struct {
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Image  string `json:"image"` // base64编码的图片
}

// AnswerItem 答案项
type AnswerItem struct {
	Type     string   `json:"type"`     // 题目类型
	Question string   `json:"question"` // 题目内容
	Options  []string `json:"options"`  // 选项
	Answer   []string `json:"answer"`   // 答案
}

// 校验过程可能返回类型
type HeaderError struct {
	Missing []string // 缺失字段
	Extra   []string // 多余字段
}

func (e HeaderError) Error() string {
	msgs := []string{}
	if len(e.Missing) > 0 {
		msgs = append(msgs, fmt.Sprintf("缺失字段：%v", e.Missing))
	}
	if len(e.Extra) > 0 {
		msgs = append(msgs, fmt.Sprintf("多余字段：%v", e.Extra))
	}
	return strings.Join(msgs, "; ")
}

// SearchResult 搜索结果
type SearchResult struct {
	Item            AnswerItem       `json:"item"`
	Score           float64          `json:"score"`           // 匹配度
	Matched         string           `json:"matched"`         // 匹配的文本
	QuestionMatches []int            `json:"questionMatches"` // 题目匹配位置
	OptionMatches   map[string][]int `json:"optionMatches"`   // 选项匹配位置，key为选项文本
	AnswerMatches   []int            `json:"answerMatches"`   // 答案匹配位置（不使用）
}

// FileDialogResult 文件对话框结果
type FileDialogResult struct {
	FilePath string `json:"filePath"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
}

// OCRResult OCR识别结果
type OCRResult struct {
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
	BBox       struct {
		XMin   int     `json:"xmin"`
		YMin   int     `json:"ymin"`
		XMax   int     `json:"xmax"`
		YMax   int     `json:"ymax"`
		Points [][]int `json:"points"`
	} `json:"bbox"`
}

// OCRResponse OCR响应结构
type OCRResponse struct {
	Success bool `json:"success"`
	Data    struct {
		TextCount int         `json:"text_count"`
		Results   []OCRResult `json:"results"`
	} `json:"data"`
}

// OCRService OCR服务结构
type OCRService struct {
	ServerURL string
	Client    *http.Client
}

// 全局OCR服务实例
// 全局OCR服务变量（已废弃，保留用于兼容性）

// ProcessImage 处理图片进行OCR识别
func (o *OCRService) ProcessImage(imageData []byte) ([]OCRResult, error) {
	// 将图片数据编码为base64
	base64Data := base64.StdEncoding.EncodeToString(imageData)

	// 准备请求数据
	requestData := map[string]string{
		"image": base64Data,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("编码请求数据失败: %v", err)
	}

	// 发送HTTP请求
	req, err := http.NewRequest("POST", o.ServerURL+"/ocr", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建OCR请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送OCR请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取OCR响应失败: %v", err)
	}

	// 解析JSON响应
	var ocrResp OCRResponse
	err = json.Unmarshal(body, &ocrResp)
	if err != nil {
		return nil, fmt.Errorf("解析OCR响应失败: %v", err)
	}

	if !ocrResp.Success {
		return nil, fmt.Errorf("OCR服务返回错误")
	}

	return ocrResp.Data.Results, nil
}

// OpenFileDialog 打开文件对话框
func (e *ExamService) OpenFileDialog(title string, fileType string) (FileDialogResult, error) {
	// 使用Wails v3的文件对话框API
	dialog := application.OpenFileDialog()

	// 设置标题
	dialog.SetTitle(title)

	// 设置文件过滤器
	if fileType == "csv" {
		dialog.AddFilter("CSV文件", "*.csv")
	} else if fileType == "excel" {
		dialog.AddFilter("Excel文件", "*.xlsx;*.xls")
	}

	// 允许选择所有文件类型
	dialog.AddFilter("所有文件", "*.*")

	// 确保可以选择文件
	dialog.CanChooseFiles(true)
	dialog.CanChooseDirectories(false)

	// 尝试附加到主窗口（如果可用）
	app := application.Get()
	if app != nil {
		windows := app.Window.GetAll()
		if len(windows) > 0 {
			mainWindow := windows[0]
			dialog.AttachToWindow(mainWindow)
		}
	}

	// 提示用户选择单个文件
	filePath, err := dialog.PromptForSingleSelection()
	if err != nil {
		return FileDialogResult{
			FilePath: "",
			Success:  false,
			Error:    fmt.Sprintf("打开文件对话框失败: %v", err),
		}, nil
	}

	// 如果用户取消了选择，filePath为空
	if filePath == "" {
		return FileDialogResult{
			FilePath: "",
			Success:  false,
			Error:    "用户取消了文件选择",
		}, nil
	}

	return FileDialogResult{
		FilePath: filePath,
		Success:  true,
	}, nil
}

// ReadFileContent 读取文件内容
// getEncoding 根据编码名称获取对应的编码器
func getEncoding(encodingName string) (encoding.Encoding, error) {
	switch strings.ToLower(encodingName) {
	case "utf8", "utf-8":
		return nil, nil // UTF-8是默认编码
	case "gbk", "gb2312":
		return simplifiedchinese.GBK, nil
	default:
		return nil, fmt.Errorf("不支持的编码格式: %s，仅支持UTF-8和GBK", encodingName)
	}
}

func (e *ExamService) ReadFileContent(filePath string, encoding string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("无法打开文件: %v", err)
	}
	defer file.Close()

	var reader io.Reader = file

	// 根据编码处理文件
	if encoding != "utf8" && encoding != "utf-8" {
		enc, err := getEncoding(encoding)
		if err != nil {
			return "", fmt.Errorf("编码设置错误: %v", err)
		}

		if enc != nil {
			reader = transform.NewReader(file, enc.NewDecoder())
		}
	}

	// 读取文件内容
	content, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %v", err)
	}

	return string(content), nil
}

// ParseCSVFile 解析CSV文件
func (e *ExamService) ParseCSVFile(filePath string, encoding string, optionSeparator string, answerSeparator string) ([]AnswerItem, error) {
	var answers []AnswerItem

	// 打开文件
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("无法打开文件: %v", err)
	}
	defer f.Close()

	// 解码器处理
	var reader io.Reader
	switch strings.ToLower(encoding) {
	case "gbk", "gb2312":
		reader = transform.NewReader(f, simplifiedchinese.GBK.NewDecoder())
	case "utf-8", "utf8":
		reader = f
	default:
		return nil, fmt.Errorf("不支持的编码格式: %s", encoding)
	}

	csvReader := csv.NewReader(reader)
	csvReader.TrimLeadingSpace = true

	// 读取标题行
	headers, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("读取标题行失败: %v", err)
	}

	expected := map[string]int{"类型": -1, "题目": -1, "选项": -1, "答案": -1}
	for i, h := range headers {
		if _, ok := expected[h]; ok {
			expected[h] = i
		}
	}

	// 检查缺失字段
	var missing []string
	for key, idx := range expected {
		if idx == -1 {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("缺少字段: %s", strings.Join(missing, ", "))
	}

	// 读取数据行
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("读取数据失败: %v", err)
		}

		answer := AnswerItem{
			Type:     strings.TrimSpace(record[expected["类型"]]),
			Question: strings.TrimSpace(record[expected["题目"]]),
			Options:  []string{},
			Answer:   []string{},
		}

		// 拆分选项
		optionsStr := record[expected["选项"]]
		if optionSeparator != "" {
			separator := e.parseSeparator(optionSeparator)
			answer.Options = strings.Split(optionsStr, separator)
		} else {
			answer.Options = []string{optionsStr}
		}

		// 拆分答案
		answerStr := record[expected["答案"]]
		if answerStr != "" {
			separator := e.parseSeparator(answerSeparator)
			answer.Answer = strings.Split(answerStr, separator)
		}

		answers = append(answers, answer)
	}

	return answers, nil
}

// parseSeparator 解析分隔符，支持转义字符
func (e *ExamService) parseSeparator(separator string) string {
	switch separator {
	case "\\n":
		return "\n"
	case "\\t":
		return "\t"
	case "\\r":
		return "\r"
	case "\\s":
		return " "
	default:
		return separator
	}
}

// TestOCRConnection 测试OCR连接
func (e *ExamService) TestOCRConnection(config OCRConfig) (string, error) {
	if config.URL == "" {
		return "连接失败", fmt.Errorf("未配置OCR服务URL")
	}

	// 构建健康检查URL
	healthURL := config.URL
	if !strings.HasSuffix(healthURL, "/") {
		healthURL += "/"
	}
	healthURL += "health"

	// 发送HTTP请求到健康检查端点
	req, err := http.NewRequest("GET", healthURL, nil)
	if err != nil {
		return "连接失败", fmt.Errorf("创建健康检查请求失败: %v", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "连接失败", fmt.Errorf("健康检查请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "连接失败", fmt.Errorf("读取健康检查响应失败: %v", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != 200 {
		return "连接失败", fmt.Errorf("健康检查失败，状态码: %d", resp.StatusCode)
	}

	// 尝试解析JSON响应
	var healthResp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(body, &healthResp); err == nil {
		if healthResp.Success {
			return "连接成功", nil
		} else {
			return "连接失败", fmt.Errorf("OCR服务报告错误: %s", healthResp.Message)
		}
	}

	// 如果无法解析JSON，但HTTP状态码是200，也认为连接成功
	return "连接成功", nil
}

// TestLocalOCR 测试本地OCR功能
func (e *ExamService) TestLocalOCR() (string, error) {
	// 使用默认的本地OCR服务URL
	defaultURL := "http://127.0.0.1:8080"

	// 读取test.png文件
	imagePath := "test.png"
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("读取图像文件失败: %v", err)
	}

	// 使用新的OCR服务处理图像
	result, err := e.performOCRWithURL(imageData, defaultURL)
	if err != nil {
		return "", fmt.Errorf("OCR处理失败: %v", err)
	}

	if result == "" {
		return "未检测到任何文本内容", nil
	}

	return fmt.Sprintf("OCR处理完成，识别结果：\n%s", result), nil
}

// TakeScreenshot 截取屏幕
func (e *ExamService) TakeScreenshot() (string, error) {
	var cmd *exec.Cmd
	var tempFile string

	// 根据操作系统选择不同的截图命令
	switch runtime.GOOS {
	case "darwin": // macOS
		tempFile = "/tmp/screenshot.png"
		cmd = exec.Command("screencapture", "-x", "-t", "png", tempFile)
	case "windows": // Windows
		tempFile = filepath.Join(os.TempDir(), "screenshot.png")
		cmd = exec.Command("powershell", "-Command", "Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing; $screen = [System.Windows.Forms.Screen]::PrimaryScreen; $bitmap = New-Object System.Drawing.Bitmap $screen.Bounds.Width, $screen.Bounds.Height; $graphics = [System.Drawing.Graphics]::FromImage($bitmap); $graphics.CopyFromScreen($screen.Bounds.X, $screen.Bounds.Y, 0, 0, $screen.Bounds.Size); $bitmap.Save('"+tempFile+"', [System.Drawing.Imaging.ImageFormat]::Png); $graphics.Dispose(); $bitmap.Dispose()")
	case "linux": // Linux
		tempFile = "/tmp/screenshot.png"
		cmd = exec.Command("import", "-window", "root", tempFile)
	default:
		return "", fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("截图失败: %v", err)
	}

	// 读取截图文件
	imageData, err := os.ReadFile(tempFile)
	if err != nil {
		return "", fmt.Errorf("读取截图文件失败: %v", err)
	}

	// 清理临时文件
	os.Remove(tempFile)

	// 转换为base64编码
	base64Data := base64.StdEncoding.EncodeToString(imageData)

	// 返回data URL格式
	return "data:image/png;base64," + base64Data, nil
}

// TakeScreenshotWithWindowControl 带窗口控制的截图
func (e *ExamService) TakeScreenshotWithWindowControl() (string, error) {
	// 获取应用实例
	app := application.Get()
	if app == nil {
		return "", fmt.Errorf("无法获取应用实例")
	}

	// 获取所有窗口
	windows := app.Window.GetAll()
	if len(windows) == 0 {
		return "", fmt.Errorf("没有找到窗口")
	}

	window := windows[0]

	// 1. 隐藏窗口
	window.Minimise()

	// 2. 等待一小段时间确保窗口完全隐藏
	time.Sleep(500 * time.Millisecond)

	// 3. 截取屏幕
	screenshot, err := e.TakeScreenshot()
	if err != nil {
		// 即使截图失败也要恢复窗口
		window.Restore()
		return "", err
	}

	// 4. 恢复窗口
	window.Restore()

	return screenshot, nil
}

// SelectArea 选择截图区域
func (e *ExamService) SelectArea(screenshotData string) (ScreenshotArea, error) {
	// 这个函数现在主要用于接收前端已经裁剪好的图片
	// 前端会直接传递裁剪后的图片数据
	area := ScreenshotArea{
		X:      0, // 裁剪后的图片，坐标从0开始
		Y:      0,
		Width:  0, // 宽度和高度会在前端设置
		Height: 0,
		Image:  screenshotData, // 这里应该是裁剪后的图片
	}
	return area, nil
}

// PerformOCR 执行OCR识别
func (e *ExamService) PerformOCR(area ScreenshotArea, config OCRConfig) (string, error) {
	if area.Image == "" {
		return "", fmt.Errorf("没有截图数据")
	}

	// 解码base64图片数据
	imageData, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(area.Image, "data:image/png;base64,"))
	if err != nil {
		return "", fmt.Errorf("图片解码失败: %v", err)
	}

	// 解码PNG图片
	img, err := png.Decode(bytes.NewReader(imageData))
	if err != nil {
		return "", fmt.Errorf("图片解码失败: %v", err)
	}

	// 如果指定了区域，裁剪图片
	if area.Width > 0 && area.Height > 0 {
		bounds := img.Bounds()
		if area.X+area.Width <= bounds.Dx() && area.Y+area.Height <= bounds.Dy() {
			img = img.(interface {
				SubImage(r image.Rectangle) image.Image
			}).SubImage(image.Rect(area.X, area.Y, area.X+area.Width, area.Y+area.Height))
		}
	}

	// 重新编码为PNG
	var buf bytes.Buffer
	err = png.Encode(&buf, img)
	if err != nil {
		return "", fmt.Errorf("图片编码失败: %v", err)
	}

	// 使用配置的OCR服务URL进行识别
	if config.URL != "" {
		return e.performOCRWithURL(buf.Bytes(), config.URL)
	}

	// 如果没有配置OCR URL，返回模拟结果
	return "这是一个模拟的OCR识别结果", nil
}

// performOCRWithURL 使用指定URL的OCR服务进行识别
func (e *ExamService) performOCRWithURL(imageData []byte, serverURL string) (string, error) {
	// 将图片数据编码为base64
	base64Data := base64.StdEncoding.EncodeToString(imageData)

	// 准备请求数据
	requestData := map[string]string{
		"image": base64Data,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return "", fmt.Errorf("编码请求数据失败: %v", err)
	}

	// 构建OCR请求URL
	ocrURL := serverURL
	if !strings.HasSuffix(ocrURL, "/") {
		ocrURL += "/"
	}
	ocrURL += "ocr"

	// 发送HTTP请求
	req, err := http.NewRequest("POST", ocrURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建OCR请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送OCR请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取OCR响应失败: %v", err)
	}

	// 解析JSON响应
	var ocrResp OCRResponse
	err = json.Unmarshal(body, &ocrResp)
	if err != nil {
		return "", fmt.Errorf("解析OCR响应失败: %v", err)
	}

	if !ocrResp.Success {
		return "", fmt.Errorf("OCR服务返回错误")
	}

	// 只保留文字内容，合并所有识别结果
	var allText strings.Builder
	for i, result := range ocrResp.Data.Results {
		if i > 0 {
			allText.WriteString(" ")
		}
		allText.WriteString(strings.TrimSpace(result.Text))
	}

	return allText.String(), nil
}

// performOnlineOCR 使用在线OCR服务
func (e *ExamService) performOnlineOCR(imageData []byte, config OCRConfig) (string, error) {
	// 准备表单数据
	formData := url.Values{}
	formData.Set("apikey", config.APIKey)
	formData.Set("language", "chs")
	formData.Set("isOverlayRequired", "false")
	formData.Set("filetype", "png")
	formData.Set("detectOrientation", "true")

	// 创建multipart表单
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// 添加文件
	part, err := writer.CreateFormFile("file", "screenshot.png")
	if err != nil {
		return "", fmt.Errorf("创建表单失败: %v", err)
	}
	_, err = part.Write(imageData)
	if err != nil {
		return "", fmt.Errorf("写入图片数据失败: %v", err)
	}

	// 添加其他参数
	for key, values := range formData {
		for _, value := range values {
			err := writer.WriteField(key, value)
			if err != nil {
				return "", fmt.Errorf("写入表单字段失败: %v", err)
			}
		}
	}

	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("关闭表单失败: %v", err)
	}

	// 发送HTTP请求
	req, err := http.NewRequest("POST", config.URL, &buf)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	// 解析JSON响应
	var result struct {
		ParsedResults []struct {
			ParsedText string `json:"ParsedText"`
		} `json:"ParsedResults"`
		ErrorMessage string `json:"ErrorMessage"`
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	if result.ErrorMessage != "" {
		return "", fmt.Errorf("OCR服务错误: %s", result.ErrorMessage)
	}

	if len(result.ParsedResults) == 0 {
		return "", fmt.Errorf("没有识别到文本")
	}

	return strings.TrimSpace(result.ParsedResults[0].ParsedText), nil
}

// normalizeText 标准化文本，移除或替换特殊字符以提高匹配率
func (e *ExamService) normalizeText(text string) string {
	// 移除常见的标点符号和特殊字符，但保留中文字符
	// 这些字符在OCR识别中经常出现，但在语义匹配时应该被忽略
	replacer := strings.NewReplacer(
		"(", "", ")", "", "[", "", "]", "", "{", "", "}", "",
		"（", "", "）", "", "【", "", "】", "", "《", "", "》", "",
		"\"", "", "'", "", "`", "", "~", "", "!", "", "@", "",
		"#", "", "$", "", "%", "", "^", "", "&", "", "*", "",
		"+", "", "=", "", "|", "", "\\", "", "/", "", "?", "",
		"<", "", ">", "", ",", "", ".", "", ";", "", ":", "",
		"、", "", "，", "", "。", "", "；", "", "：", "", "！", "",
		"？", "", "…", "", "—", "", "－", "", "·", "", "·", "",
		"　", " ", "  ", " ", // 多个空格替换为单个空格
	)

	normalized := replacer.Replace(text)

	// 移除多余的空格
	normalized = strings.TrimSpace(normalized)

	// 将多个连续空格替换为单个空格
	for strings.Contains(normalized, "  ") {
		normalized = strings.ReplaceAll(normalized, "  ", " ")
	}

	return normalized
}

// SearchAnswers 搜索答案
// AccuracyFilters 准确度筛选参数
type AccuracyFilters struct {
	High   bool `json:"high"`   // 高准确率 (≥80%)
	Medium bool `json:"medium"` // 中准确率 (50%-79%)
	Low    bool `json:"low"`    // 低准确率 (<50%)
}

func (e *ExamService) SearchAnswers(answers []AnswerItem, query string, filters AccuracyFilters) ([]SearchResult, error) {
	results := []SearchResult{}

	// 预处理查询文本，移除特殊字符
	normalizedQuery := e.normalizeText(query)
	normalizedQuery = strings.ToLower(strings.TrimSpace(normalizedQuery))

	// 如果查询为空，返回所有答案
	if normalizedQuery == "" {
		log.Println("查询为空，返回所有答案")
		for _, answer := range answers {
			results = append(results, SearchResult{
				Item:            answer,
				Score:           0.5, // 给予中等匹配度
				Matched:         "全部结果",
				QuestionMatches: []int{},
				OptionMatches:   make(map[string][]int),
				AnswerMatches:   []int{},
			})
		}
		return results, nil
	}

	// 如果答案数据为空，返回空结果
	if len(answers) == 0 {
		log.Println("答案数据为空")
		return results, nil
	}

	// 记录所有可能的匹配结果
	allPossibleMatches := []SearchResult{}

	for _, answer := range answers {
		question := answer.Question
		// 预处理题目文本
		normalizedQuestion := e.normalizeText(question)
		questionLower := strings.ToLower(normalizedQuestion)
		score := 0.0
		matched := ""
		maxScore := 0.0

		// 分别存储各字段的匹配位置
		questionMatches := []int{}
		optionMatches := make(map[string][]int) // 为每个选项单独存储匹配位置
		answerMatches := []int{}                // 答案不需要高亮，保持空数组

		// 计算题目重合度（使用标准化后的文本进行匹配）
		questionScore, _ := e.calculateOverlapScore(normalizedQuery, questionLower)
		questionMatches = e.calculateMatchesForOriginalText(question, normalizedQuery)
		if questionScore > maxScore {
			maxScore = questionScore
			matched = normalizedQuery
		}

		// 计算答案重合度
		for _, ans := range answer.Answer {
			normalizedAns := e.normalizeText(ans)
			ansLower := strings.ToLower(normalizedAns)
			ansScore, _ := e.calculateOverlapScore(normalizedQuery, ansLower)
			ansMatches := e.calculateMatchesForOriginalText(ans, normalizedQuery)
			if ansScore > maxScore {
				maxScore = ansScore
				matched = normalizedQuery
			}
			// 合并所有答案的匹配位置
			answerMatches = append(answerMatches, ansMatches...)
		}

		// 计算选项重合度
		for _, option := range answer.Options {
			normalizedOption := e.normalizeText(option)
			optionLower := strings.ToLower(normalizedOption)
			optionScore, _ := e.calculateOverlapScore(normalizedQuery, optionLower)
			optionMatchesForThis := e.calculateMatchesForOriginalText(option, normalizedQuery)
			optionScore = optionScore * 0.8 // 选项权重稍低
			if optionScore > maxScore {
				maxScore = optionScore
				matched = "选项匹配: " + normalizedQuery
			}
			// 为每个选项单独存储匹配位置
			optionMatches[option] = optionMatchesForThis
		}

		score = maxScore

		// 记录所有可能的匹配结果，包括低匹配度的
		if score >= 0 {
			// 限制分数不超过1.0
			if score > 1.0 {
				score = 1.0
			}

			// 根据准确度筛选
			shouldInclude := false

			// 如果所有过滤器都为false，显示所有结果
			if !filters.High && !filters.Medium && !filters.Low {
				shouldInclude = true
			} else {
				// 否则按过滤器筛选
				if score >= 0.8 {
					shouldInclude = filters.High
				} else if score >= 0.5 {
					shouldInclude = filters.Medium
				} else {
					shouldInclude = filters.Low
				}
			}

			if shouldInclude {
				log.Printf("搜索结果: 题目='%s', 分数=%.2f, 题目匹配=%v, 选项匹配=%v, 答案匹配=%v",
					answer.Question, score, questionMatches, optionMatches, answerMatches)
				log.Printf("filters: %v", filters)
				allPossibleMatches = append(allPossibleMatches, SearchResult{
					Item:            answer,
					Score:           score,
					Matched:         matched,
					QuestionMatches: questionMatches,
					OptionMatches:   optionMatches,
					AnswerMatches:   answerMatches,
				})
			}
		}
	}

	// 按匹配度排序
	sort.Slice(allPossibleMatches, func(i, j int) bool {
		return allPossibleMatches[i].Score > allPossibleMatches[j].Score
	})

	return allPossibleMatches, nil
}

// calculateOverlapScore 计算重合度分数 - 使用智能匹配算法
func (e *ExamService) calculateOverlapScore(query, text string) (float64, []int) {
	if query == "" || text == "" {
		return 0.0, nil
	}

	// 完全匹配 - 100%匹配度
	if query == text {
		// 生成所有字符位置的索引
		matches := []int{}
		for i := 0; i < utf8.RuneCountInString(query); i++ {
			matches = append(matches, i)
		}
		return 1.0, matches
	}

	// 连续包含匹配 - 给予高匹配度
	if strings.Contains(text, query) {
		start := strings.Index(text, query)

		// 将字节位置转换为字符位置
		charStart := utf8.RuneCountInString(text[:start])
		charLen := utf8.RuneCountInString(query)

		// 生成匹配位置的字符索引
		matches := []int{}
		for i := charStart; i < charStart+charLen; i++ {
			matches = append(matches, i)
		}

		// 计算匹配率：匹配字符数 / 目标文本总字符数
		textLen := utf8.RuneCountInString(text)
		matchRate := float64(charLen) / float64(textLen)

		// 包含匹配给予90-95%的匹配度，但不超过95%
		score := 0.9 + matchRate*0.05
		if score > 0.95 {
			score = 0.95
		}

		return score, matches
	}

	// 目标文本包含在查询文本中
	if strings.Contains(query, text) {
		// 生成所有字符位置的索引
		matches := []int{}
		textLen := utf8.RuneCountInString(text)
		for i := 0; i < textLen; i++ {
			matches = append(matches, i)
		}

		// 完全匹配，给予95%的匹配度
		return 0.95, matches
	}

	// 使用智能匹配算法
	return e.calculateSmartSimilarity(query, text)
}

// calculateSmartSimilarity 智能相似度计算
func (e *ExamService) calculateSmartSimilarity(query, text string) (float64, []int) {
	// 1. 首先检查是否有任何共同的关键词
	commonWords := e.findCommonWords(query, text)
	if len(commonWords) == 0 {
		// 没有共同关键词，尝试使用编辑距离作为备选方案
		editDistance := e.calculateEditDistance([]rune(query), []rune(text))
		maxPossibleDistance := max(len(query), len(text))
		editSimilarity := 1.0 - float64(editDistance)/float64(maxPossibleDistance)

		// 降低阈值，允许更多可能的匹配
		if editSimilarity > 0.3 {
			matches := e.calculateSimpleMatches([]rune(query), []rune(text))
			return editSimilarity * 0.6, matches // 降低权重
		}
		return 0, nil
	}

	// 2. 计算编辑距离相似度
	editDistance := e.calculateEditDistance([]rune(query), []rune(text))
	maxPossibleDistance := max(len(query), len(text))
	editSimilarity := 1.0 - float64(editDistance)/float64(maxPossibleDistance)

	// 3. 计算关键词匹配度
	keywordSimilarity := e.calculateKeywordSimilarity(query, text)

	// 4. 计算字符匹配度
	charSimilarity := e.calculateCharSimilarity(query, text)

	// 5. 综合评分：编辑距离20%，关键词匹配50%，字符匹配30%
	similarity := editSimilarity*0.2 + keywordSimilarity*0.5 + charSimilarity*0.3

	// 6. 调整阈值，允许更多可能的匹配
	if similarity < 0.1 {
		return 0.0, nil
	}

	// 7. 计算匹配位置
	matches := e.calculateSimpleMatches([]rune(query), []rune(text))

	return similarity, matches
}

// calculateCharSimilarity 计算字符级别的相似度
func (e *ExamService) calculateCharSimilarity(query, text string) float64 {
	if query == "" || text == "" {
		return 0.0
	}

	// 将字符串转换为字符数组
	queryChars := []rune(query)
	textChars := []rune(text)

	// 计算共同字符的数量
	commonChars := 0
	queryCharSet := make(map[rune]int)
	textCharSet := make(map[rune]int)

	// 统计查询文本中的字符
	for _, char := range queryChars {
		queryCharSet[char]++
	}

	// 统计目标文本中的字符
	for _, char := range textChars {
		textCharSet[char]++
	}

	// 计算共同字符数
	for char, queryCount := range queryCharSet {
		if textCount, exists := textCharSet[char]; exists {
			// 取两个文本中该字符出现次数的最小值
			commonChars += min(queryCount, textCount)
		}
	}

	// 计算相似度：共同字符数 / 总字符数
	totalChars := len(queryChars) + len(textChars)
	if totalChars == 0 {
		return 0.0
	}

	similarity := float64(commonChars*2) / float64(totalChars)
	return similarity
}

// findCommonWords 查找共同的关键词
func (e *ExamService) findCommonWords(s1, s2 string) []string {
	// 将文本分割为单词
	words1 := e.extractWords(s1)
	words2 := e.extractWords(s2)

	// 找到共同的单词
	common := make([]string, 0)
	wordSet := make(map[string]bool)

	for _, word := range words1 {
		wordSet[word] = true
	}

	for _, word := range words2 {
		if wordSet[word] && len(word) > 1 { // 忽略单字符单词
			common = append(common, word)
		}
	}

	return common
}

// extractWords 提取文本中的单词
func (e *ExamService) extractWords(text string) []string {
	// 移除标点符号和特殊字符
	normalized := e.normalizeText(text)

	// 按空格分割
	words := strings.Fields(normalized)

	// 过滤掉太短的单词
	result := make([]string, 0)
	for _, word := range words {
		if len(word) > 1 {
			result = append(result, strings.ToLower(word))
		}
	}

	return result
}

// calculateKeywordSimilarity 计算关键词相似度
func (e *ExamService) calculateKeywordSimilarity(s1, s2 string) float64 {
	words1 := e.extractWords(s1)
	words2 := e.extractWords(s2)

	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	// 计算共同单词的数量
	commonWords := e.findCommonWords(s1, s2)
	commonCount := len(commonWords)

	// 计算相似度：共同单词数 / 总单词数
	totalWords := len(words1) + len(words2) - commonCount
	if totalWords == 0 {
		return 0.0
	}

	similarity := float64(commonCount) / float64(totalWords)

	// 给予共同单词的权重奖励，但更合理的计算方式
	if commonCount > 0 {
		// 计算共同单词在各自文本中的覆盖率
		coverage1 := float64(commonCount) / float64(len(words1))
		coverage2 := float64(commonCount) / float64(len(words2))

		// 使用几何平均数来平衡两个覆盖率
		balancedCoverage := (coverage1 + coverage2) / 2.0

		// 综合相似度和覆盖率
		similarity = (similarity + balancedCoverage) / 2.0

		if similarity > 1.0 {
			similarity = 1.0
		}
	}

	return similarity
}

// calculateEditDistance 计算编辑距离
func (e *ExamService) calculateEditDistance(query, text []rune) int {
	lenQuery := len(query)
	lenText := len(text)

	// 创建DP表
	dp := make([][]int, lenQuery+1)
	for i := range dp {
		dp[i] = make([]int, lenText+1)
	}

	// 初始化第一行和第一列
	for i := 0; i <= lenQuery; i++ {
		dp[i][0] = i
	}
	for j := 0; j <= lenText; j++ {
		dp[0][j] = j
	}

	// 填充DP表
	for i := 1; i <= lenQuery; i++ {
		for j := 1; j <= lenText; j++ {
			if query[i-1] == text[j-1] {
				dp[i][j] = dp[i-1][j-1]
			} else {
				dp[i][j] = min(dp[i-1][j], min(dp[i][j-1], dp[i-1][j-1])) + 1
			}
		}
	}

	return dp[lenQuery][lenText]
}

// calculateSimpleMatches 计算简化的匹配位置
func (e *ExamService) calculateSimpleMatches(query, text []rune) []int {
	matches := []int{}

	// 将rune切片转换为字符串进行匹配
	queryStr := string(query)
	textStr := string(text)

	// 情况1：查询文本包含在目标文本中
	if strings.Contains(textStr, queryStr) {
		// 找到匹配的起始位置
		start := strings.Index(textStr, queryStr)

		// 将字节位置转换为字符位置
		charStart := utf8.RuneCountInString(textStr[:start])
		charLen := utf8.RuneCountInString(queryStr)

		// 生成字符位置索引
		for i := charStart; i < charStart+charLen; i++ {
			matches = append(matches, i)
		}
		return matches
	}

	// 情况2：目标文本包含在查询文本中
	if strings.Contains(queryStr, textStr) {
		// 生成字符位置索引（相对于目标文本）
		charLen := utf8.RuneCountInString(textStr)
		for i := 0; i < charLen; i++ {
			matches = append(matches, i)
		}
		return matches
	}

	// 情况3：查找最长公共子串
	commonSubstr := e.findLongestCommonSubstring(queryStr, textStr)
	if len(commonSubstr) > 0 {
		// 在目标文本中找到公共子串的位置
		start := strings.Index(textStr, commonSubstr)
		if start != -1 {
			charStart := utf8.RuneCountInString(textStr[:start])
			charLen := utf8.RuneCountInString(commonSubstr)

			// 生成字符位置索引
			for i := charStart; i < charStart+charLen; i++ {
				matches = append(matches, i)
			}
		}
	}

	return matches
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// contains 检查切片是否包含某个值
func contains(slice []int, value int) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// max 返回两个整数中的较大值
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// mapMatchesToOriginalText 将小写文本的匹配位置映射到原始文本
func (e *ExamService) mapMatchesToOriginalText(original, lower string, matches []int) []int {
	if len(matches) == 0 {
		return matches
	}

	log.Printf("映射匹配位置: 原始文本='%s', 小写文本='%s', 匹配位置=%v", original, lower, matches)

	// 对于简单的ASCII文本，位置通常是相同的
	// 但对于包含大写字母的文本，需要调整
	originalMatches := []int{}

	for _, match := range matches {
		if match < len(lower) {
			// 找到在原始文本中对应的位置
			originalPos := e.findCorrespondingPosition(original, lower, match)
			if originalPos >= 0 {
				originalMatches = append(originalMatches, originalPos)
			}
		}
	}

	log.Printf("映射后的匹配位置: %v", originalMatches)
	return originalMatches
}

// findCorrespondingPosition 找到原始文本中对应的位置
func (e *ExamService) findCorrespondingPosition(original, lower string, lowerPos int) int {
	if lowerPos >= len(lower) {
		return -1
	}

	// 对于包含大写字母的情况，需要找到对应的原始位置
	// 例如：原始="Vue.js"，小写="vue.js"，小写位置2对应原始位置0
	lowerPosCount := 0

	for i := range original {
		if lowerPosCount == lowerPos {
			return i
		}
		lowerPosCount++
	}

	// 如果没找到，返回-1
	return -1
}

// 简化匹配位置计算，直接基于原始文本计算
func (e *ExamService) calculateMatchesForOriginalText(originalText, query string) []int {
	matches := []int{}

	// 标准化原始文本和查询文本
	normalizedOriginal := e.normalizeText(originalText)
	normalizedQuery := e.normalizeText(query)

	originalLower := strings.ToLower(normalizedOriginal)
	queryLower := strings.ToLower(normalizedQuery)

	// 情况1：查询文本包含在目标文本中
	if strings.Contains(originalLower, queryLower) {
		start := strings.Index(originalLower, queryLower)

		// 将字节位置转换为字符位置
		charStart := utf8.RuneCountInString(originalLower[:start])
		charLen := utf8.RuneCountInString(queryLower)

		// 将标准化文本的位置映射回原始文本的位置
		originalMatches := e.mapNormalizedPositionsToOriginal(originalText, normalizedOriginal, charStart, charLen)
		matches = originalMatches
		return matches
	}

	// 情况2：目标文本包含在查询文本中
	if strings.Contains(queryLower, originalLower) {
		// 目标文本完全匹配，返回所有位置
		charLen := utf8.RuneCountInString(originalLower)
		originalMatches := e.mapNormalizedPositionsToOriginal(originalText, normalizedOriginal, 0, charLen)
		matches = originalMatches
		return matches
	}

	// 情况3：查找最长公共子串
	commonSubstr := e.findLongestCommonSubstring(originalLower, queryLower)
	if len(commonSubstr) > 0 {
		// 在目标文本中找到公共子串的位置
		start := strings.Index(originalLower, commonSubstr)
		if start != -1 {
			charStart := utf8.RuneCountInString(originalLower[:start])
			charLen := utf8.RuneCountInString(commonSubstr)

			// 将标准化文本的位置映射回原始文本的位置
			originalMatches := e.mapNormalizedPositionsToOriginal(originalText, normalizedOriginal, charStart, charLen)
			matches = originalMatches
		}
	}

	return matches
}

// mapNormalizedPositionsToOriginal 将标准化文本的位置映射回原始文本的位置
func (e *ExamService) mapNormalizedPositionsToOriginal(originalText, normalizedText string, normalizedStart, normalizedLen int) []int {
	matches := []int{}

	// 将原始文本和标准化文本都转换为字符数组
	originalChars := []rune(originalText)
	normalizedChars := []rune(normalizedText)

	// 创建标准化文本到原始文本的位置映射
	normalizedToOriginal := make([]int, len(normalizedChars))

	// 构建位置映射关系
	normalizedPos := 0
	for originalPos, char := range originalChars {
		// 检查这个字符在标准化后是否保留
		normalizedChar := e.normalizeChar(char)
		if normalizedChar != 0 {
			if normalizedPos < len(normalizedToOriginal) {
				normalizedToOriginal[normalizedPos] = originalPos
				normalizedPos++
			}
		}
		// 如果字符被移除，跳过
	}

	// 将标准化文本的匹配位置映射回原始文本位置
	for i := normalizedStart; i < normalizedStart+normalizedLen; i++ {
		if i >= 0 && i < len(normalizedToOriginal) {
			originalPos := normalizedToOriginal[i]
			if originalPos >= 0 && originalPos < len(originalChars) {
				matches = append(matches, originalPos)
			}
		}
	}

	return matches
}

// normalizeChar 标准化单个字符，返回标准化后的字符，如果字符被移除则返回0
func (e *ExamService) normalizeChar(char rune) rune {
	// 移除标点符号和特殊字符，与normalizeText函数保持一致
	specialChars := map[rune]bool{
		'(': true, ')': true, '[': true, ']': true, '{': true, '}': true,
		'（': true, '）': true, '【': true, '】': true, '《': true, '》': true,
		'"': true, '\'': true, '`': true, '~': true, '!': true, '@': true,
		'#': true, '$': true, '%': true, '^': true, '&': true, '*': true,
		'+': true, '=': true, '|': true, '\\': true, '/': true, '?': true,
		'<': true, '>': true, ',': true, '.': true, ';': true, ':': true,
		'、': true, '，': true, '。': true, '；': true, '：': true, '！': true,
		'？': true, '…': true, '—': true, '－': true, '·': true, '　': true,
	}

	if specialChars[char] {
		return 0 // 返回0表示字符被移除
	}
	return char
}

// NextQuestion 下一题功能
func (e *ExamService) NextQuestion(area ScreenshotArea, config OCRConfig) (string, error) {
	// 1. 重新截图
	screenshot, err := e.TakeScreenshot()
	if err != nil {
		return "", err
	}

	// 2. 执行OCR识别
	area.Image = screenshot
	ocrResult, err := e.PerformOCR(area, config)
	if err != nil {
		return "", err
	}

	return ocrResult, nil
}

// findLongestCommonSubstring 查找两个字符串的最长公共子串
func (e *ExamService) findLongestCommonSubstring(s1, s2 string) string {
	if len(s1) == 0 || len(s2) == 0 {
		return ""
	}

	// 使用动态规划算法查找最长公共子串
	dp := make([][]int, len(s1)+1)
	for i := range dp {
		dp[i] = make([]int, len(s2)+1)
	}

	maxLen := 0
	endPos := 0

	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			if s1[i-1] == s2[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
				if dp[i][j] > maxLen {
					maxLen = dp[i][j]
					endPos = i - 1
				}
			}
		}
	}

	if maxLen == 0 {
		return ""
	}

	startPos := endPos - maxLen + 1
	return s1[startPos : endPos+1]
}

// HideWindow 隐藏应用窗口
func (e *ExamService) HideWindow() error {
	// 获取应用实例
	app := application.Get()
	if app == nil {
		return fmt.Errorf("无法获取应用实例")
	}

	// 获取所有窗口
	windows := app.Window.GetAll()
	if len(windows) == 0 {
		return fmt.Errorf("没有找到窗口")
	}

	// 隐藏第一个窗口（主窗口）
	// 使用更安全的方式隐藏窗口
	windows[0].Minimise()
	return nil
}

// ShowWindow 显示应用窗口
func (e *ExamService) ShowWindow() error {
	// 获取应用实例
	app := application.Get()
	if app == nil {
		return fmt.Errorf("无法获取应用实例")
	}

	// 获取所有窗口
	windows := app.Window.GetAll()
	if len(windows) == 0 {
		return fmt.Errorf("没有找到窗口")
	}

	// 显示第一个窗口（主窗口）
	// 使用更安全的方式显示窗口
	windows[0].Restore()
	return nil
}

// 全局变量存储答案数据
var globalAnswers []AnswerItem

// SetGlobalAnswers 设置全局答案数据
func (e *ExamService) SetGlobalAnswers(answers []AnswerItem) {
	globalAnswers = answers
}

// GetGlobalAnswers 获取全局答案数据
func (e *ExamService) GetGlobalAnswers() []AnswerItem {
	return globalAnswers
}

// SearchRequest HTTP搜索请求结构
type SearchRequest struct {
	Query   string        `json:"query"`
	Filters SearchFilters `json:"filters"`
}

type SearchFilters struct {
	AccuracyFilters AccuracyFilters `json:"accuracyFilters"`
}

// SearchResponse HTTP搜索响应结构
type SearchResponse struct {
	Success bool           `json:"success"`
	Message string         `json:"message,omitempty"`
	Results []SearchResult `json:"results,omitempty"`
}

// ParseCSVRequest HTTP CSV解析请求结构
type ParseCSVRequest struct {
	FilePath        string `json:"filePath"`
	Encoding        string `json:"encoding"`
	OptionSeparator string `json:"optionSeparator"`
	AnswerSeparator string `json:"answerSeparator"`
}

// ParseCSVResponse HTTP CSV解析响应结构
type ParseCSVResponse struct {
	Success bool         `json:"success"`
	Message string       `json:"message,omitempty"`
	Results []AnswerItem `json:"results,omitempty"`
}

// SetGlobalAnswersRequest HTTP设置全局答案请求结构
type SetGlobalAnswersRequest struct {
	Answers []AnswerItem `json:"answers"`
}

// SetGlobalAnswersResponse HTTP设置全局答案响应结构
type SetGlobalAnswersResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// GetGlobalAnswersResponse HTTP获取全局答案响应结构
type GetGlobalAnswersResponse struct {
	Success bool         `json:"success"`
	Message string       `json:"message,omitempty"`
	Answers []AnswerItem `json:"answers,omitempty"`
}

// handleParseCSV 处理HTTP CSV解析请求
func handleParseCSV(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// 处理预检请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 只允许POST方法
	if r.Method != "POST" {
		http.Error(w, "只支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求体
	var req ParseCSVRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "请求体解析失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 创建ExamService实例
	examService := &ExamService{}

	// 调用ParseCSVFile方法
	results, err := examService.ParseCSVFile(req.FilePath, req.Encoding, req.OptionSeparator, req.AnswerSeparator)
	if err != nil {
		response := ParseCSVResponse{
			Success: false,
			Message: "CSV解析失败: " + err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// 返回解析结果
	response := ParseCSVResponse{
		Success: true,
		Results: results,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleSearch 处理HTTP搜索请求
func handleSearch(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// 处理预检请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 只允许POST方法
	if r.Method != "POST" {
		http.Error(w, "只支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求体
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "请求体解析失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 创建ExamService实例
	examService := &ExamService{}

	// 使用全局答案数据进行搜索
	log.Printf("req %v", req)
	results, err := examService.SearchAnswers(globalAnswers, req.Query, req.Filters.AccuracyFilters)
	if err != nil {
		response := SearchResponse{
			Success: false,
			Message: "搜索失败: " + err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// 返回搜索结果
	response := SearchResponse{
		Success: true,
		Results: results,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleSetGlobalAnswers 处理HTTP设置全局答案请求
func handleSetGlobalAnswers(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// 处理预检请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 只允许POST方法
	if r.Method != "POST" {
		http.Error(w, "只支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求体
	var req SetGlobalAnswersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "请求体解析失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 创建ExamService实例
	examService := &ExamService{}

	// 调用SetGlobalAnswers方法
	examService.SetGlobalAnswers(req.Answers)

	// 返回设置结果
	response := SetGlobalAnswersResponse{
		Success: true,
		Message: fmt.Sprintf("成功设置 %d 条答案", len(req.Answers)),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetGlobalAnswers 处理HTTP获取全局答案请求
func handleGetGlobalAnswers(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// 处理预检请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 只允许GET方法
	if r.Method != "GET" {
		http.Error(w, "只支持GET方法", http.StatusMethodNotAllowed)
		return
	}

	// 创建ExamService实例
	examService := &ExamService{}

	// 调用GetGlobalAnswers方法
	answers := examService.GetGlobalAnswers()

	// 返回获取结果
	response := GetGlobalAnswersResponse{
		Success: true,
		Answers: answers,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// TestOCRRequest HTTP OCR测试请求结构
type TestOCRRequest struct {
	Config OCRConfig `json:"config"`
}

// TestOCRResponse HTTP OCR测试响应结构
type TestOCRResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Result  string `json:"result,omitempty"`
}

// ScreenshotResponse HTTP截图响应结构
type ScreenshotResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Image   string `json:"image,omitempty"`
}

// PerformOCRRequest HTTP执行OCR请求结构
type PerformOCRRequest struct {
	Area   ScreenshotArea `json:"area"`
	Config OCRConfig      `json:"config"`
}

// PerformOCRResponse HTTP执行OCR响应结构
type PerformOCRResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Result  string `json:"result,omitempty"`
}

// handleTestOCR 处理HTTP OCR测试请求
func handleTestOCR(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// 处理预检请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 只允许POST方法
	if r.Method != "POST" {
		http.Error(w, "只支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求体
	var req TestOCRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "请求体解析失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 创建ExamService实例
	examService := &ExamService{}

	// 调用TestOCRConnection方法
	result, err := examService.TestOCRConnection(req.Config)
	if err != nil {
		response := TestOCRResponse{
			Success: false,
			Message: "OCR测试失败: " + err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// 返回测试结果
	response := TestOCRResponse{
		Success: true,
		Result:  result,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleTakeScreenshot 处理HTTP截图请求
func handleTakeScreenshot(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// 处理预检请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 只允许POST方法
	if r.Method != "POST" {
		http.Error(w, "只支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	// 创建ExamService实例
	examService := &ExamService{}

	// 调用TakeScreenshotWithWindowControl方法
	image, err := examService.TakeScreenshotWithWindowControl()
	if err != nil {
		response := ScreenshotResponse{
			Success: false,
			Message: "截图失败: " + err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// 返回截图结果
	response := ScreenshotResponse{
		Success: true,
		Image:   image,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handlePerformOCR 处理HTTP执行OCR请求
func handlePerformOCR(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// 处理预检请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 只允许POST方法
	if r.Method != "POST" {
		http.Error(w, "只支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求体
	var req PerformOCRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "请求体解析失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 创建ExamService实例
	examService := &ExamService{}

	// 调用PerformOCR方法
	result, err := examService.PerformOCR(req.Area, req.Config)
	if err != nil {
		response := PerformOCRResponse{
			Success: false,
			Message: "OCR执行失败: " + err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// 返回OCR结果
	response := PerformOCRResponse{
		Success: true,
		Result:  result,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
