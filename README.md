# 考试小助手

一个基于Wails框架开发的跨平台桌面应用程序，帮助用户快速查找答案的考试助手工具。

## 功能特性

- 🖥️ 跨平台支持：Windows、macOS、Linux
- 🔍 OCR文字识别功能
- 📊 Excel和CSV文件支持
- 🎨 现代化用户界面
- ⚡ 快速搜索和匹配

## 技术栈

- **后端**: Go 1.24 + Wails v3
- **前端**: Vue 3 + Vite + TDesign
- **OCR**: 基于ONNX的轻量级OCR服务

## 开发环境要求

- Go 1.24+
- Node.js 18+
- Wails CLI

## 本地开发

### 1. 克隆项目
```bash
git clone https://github.com/your-username/exam_assistant.git
cd exam_assistant
```

### 2. 安装依赖
```bash
# 安装前端依赖
cd frontend
npm install
cd ..

# 安装Wails CLI
go install github.com/wailsapp/wails/v3/cmd/wails@latest
```

### 3. 启动OCR服务
```bash
cd ocr_minimal
chmod +x start.sh
./start.sh
```

### 4. 开发模式运行
```bash
# 终端1：启动前端开发服务器
cd frontend
npm run dev

# 终端2：启动Wails开发模式
wails dev
```

### 5. 构建应用
```bash
# 构建前端
cd frontend
npm run build
cd ..

# 构建桌面应用
wails build
```

## GitHub Actions 自动化构建

本项目使用GitHub Actions进行多平台自动化构建和发布。

### 支持的平台

- ✅ macOS Intel (x64)
- ✅ macOS Apple Silicon (ARM64)
- ✅ Windows AMD64
- ✅ Windows ARM64
- ✅ Linux AMD64
- ✅ Linux ARM64

### 触发构建

1. **推送标签触发**：推送以 `v` 开头的标签（如 `v1.0.0`）
2. **手动触发**：在GitHub仓库页面手动运行工作流

### 构建流程

1. 设置Go和Node.js环境
2. 安装前端依赖并构建
3. 安装Wails CLI
4. 为每个目标平台构建应用
5. 上传构建产物
6. 创建GitHub Release（仅标签触发时）

### 使用步骤

1. **初始化Git仓库**
```bash
git init
git add .
git commit -m "Initial commit"
```

2. **创建GitHub仓库**
   - 在GitHub上创建新仓库
   - 不要初始化README、.gitignore或license

3. **推送代码到GitHub**
```bash
git remote add origin https://github.com/your-username/exam_assistant.git
git branch -M main
git push -u origin main
```

4. **创建标签触发构建**
```bash
git tag v1.0.0
git push origin v1.0.0
```

5. **查看构建结果**
   - 在GitHub仓库页面点击"Actions"标签
   - 查看构建进度和结果
   - 构建完成后会自动创建Release

## 项目结构

```
exam_assistant/
├── .github/workflows/    # GitHub Actions配置
├── frontend/             # Vue.js前端代码
├── ocr_minimal/         # OCR服务相关文件
├── main.go              # 主程序入口
├── greetservice.go      # 服务定义
└── README.md           # 项目说明
```

## 许可证

本项目采用MIT许可证。

## 贡献

欢迎提交Issue和Pull Request！

## 更新日志

### v1.0.0
- 初始版本发布
- 支持OCR文字识别
- 支持Excel和CSV文件
- 跨平台桌面应用
