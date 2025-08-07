<script setup>
// 热更新测试 - 修改这个注释来测试热更新 - 已修复！✅
import { ref, reactive, onMounted } from 'vue'
import OCRConfig from './components/OCRConfig.vue'
import ImportConfig from './components/ImportConfig.vue'
import AreaSelector from './components/AreaSelector.vue'
import FunctionArea from './components/FunctionArea.vue'
import AnswerDisplay from './components/AnswerDisplay.vue'
import ErrorDialog from './components/ErrorDialog.vue'

// 响应式数据
const leftPanelWidth = ref(600) // 默认占50% (1200px的一半)
const currentAnswers = ref([]) // 当前导入的答案数据

// 区域选择弹窗相关
const showAreaSelector = ref(false)
const fullScreenshot = ref('')
const selectedArea = ref(null)
const isSelecting = ref(false)
const startPoint = ref({ x: 0, y: 0 })
const imageRef = ref(null)

// 组件引用
const ocrConfigRef = ref(null)
const importConfigRef = ref(null)
const areaSelectorRef = ref(null)
const functionAreaRef = ref(null)
const answerDisplayRef = ref(null)
const errorDialogRef = ref(null)

// 准确率筛选状态
const accuracyFilters = reactive({
  high: true,   // 高准确率 (≥80%)
  medium: true, // 中准确率 (50%-79%)
  low: true     // 低准确率 (<50%)
})

// 拖动调整宽度相关
const isResizing = ref(false)
const startX = ref(0)
const startWidth = ref(0)

// 开始拖动
const startResize = (e) => {
  isResizing.value = true
  startX.value = e.type === 'mousedown' ? e.clientX : e.touches[0].clientX
  startWidth.value = leftPanelWidth.value
  
  document.addEventListener('mousemove', handleResize)
  document.addEventListener('mouseup', stopResize)
  document.addEventListener('touchmove', handleResize)
  document.addEventListener('touchend', stopResize)
}

// 处理拖动
const handleResize = (e) => {
  if (!isResizing.value) return
  
  const currentX = e.type === 'mousemove' ? e.clientX : e.touches[0].clientX
  const deltaX = currentX - startX.value
  const newWidth = startWidth.value + deltaX
  
  // 限制最小和最大宽度 (30% - 70%)
  if (newWidth >= 360 && newWidth <= 840) {
    leftPanelWidth.value = newWidth
  }
}

// 停止拖动
const stopResize = () => {
  isResizing.value = false
  document.removeEventListener('mousemove', handleResize)
  document.removeEventListener('mouseup', stopResize)
  document.removeEventListener('touchmove', handleResize)
  document.removeEventListener('touchend', stopResize)
}

// 处理导入成功
const handleImportSuccess = (newAnswers) => {
  currentAnswers.value = newAnswers // 保存答案数据
  answerDisplayRef.value?.updateAnswers(newAnswers)
}

// 处理导入错误
const handleImportError = (error) => {
  console.error('导入失败:', error)
  
  // 根据错误类型显示详细的错误信息
  let errorTitle = '文件导入失败'
  let errorContent = ''
  let errorDetails = ''
  
  if (error.message.includes('无法打开CSV文件')) {
    errorTitle = 'CSV文件访问失败'
    errorContent = '无法打开CSV文件，请检查文件是否存在且可访问'
    errorDetails = '可能的原因：\n• 文件路径不正确\n• 文件权限不足\n• 文件被其他程序占用'
  } else if (error.message.includes('编码设置错误')) {
    errorTitle = '文件编码错误'
    errorContent = '文件编码设置错误，请尝试其他编码格式'
    errorDetails = '支持的编码格式：\n• UTF-8（推荐）\n• GBK（中文Windows）'
  } else if (error.message.includes('文件为空')) {
    errorTitle = '文件为空'
    errorContent = '选择的文件为空，请选择包含数据的文件'
    errorDetails = '请选择包含答案数据的文件'
  } else if (error.message.includes('文件缺少标题行')) {
    errorTitle = '文件缺少标题行'
    errorContent = '文件缺少标题行，请确保CSV文件第一行包含列名'
    errorDetails = '请确保CSV文件第一行包含以下列名：\n• 类型\n• 题目\n• 选项\n• 答案\n\n注意：列名必须包含这些关键词，顺序可以不同'
  } else if (error.message.includes('解析CSV文件失败')) {
    errorTitle = 'CSV文件解析失败'
    errorContent = 'CSV文件格式错误，请检查文件内容'
    errorDetails = '请检查：\n• CSV文件格式是否正确\n• 列分隔符是否正确\n• 数据是否完整'
  } else {
    errorTitle = '文件导入失败'
    errorContent = '文件导入过程中发生未知错误'
    errorDetails = `错误详情：${error.message}`
  }
  
  errorDialogRef.value?.showError(errorTitle, errorContent, errorDetails)
}

// 处理区域选择完成
const handleAreaSelected = (area) => {
  console.log('区域选择完成:', area)
}

// 开始区域选择
const startAreaSelection = async () => {
  try {
    // 使用HTTP服务截图
    const { takeScreenshot } = await import('./services/httpService.js')
    
    // 使用带窗口控制的截图方法
    const screenshot = await takeScreenshot()
    fullScreenshot.value = screenshot
    
    // 等待一小段时间确保窗口完全显示
    await new Promise(resolve => setTimeout(resolve, 300))
    
    // 显示区域选择器
    showAreaSelector.value = true
    
    // 初始化选择区域
    selectedArea.value = {
      x: 100,
      y: 100,
      width: 500,
      height: 300
    }
  } catch (error) {
    console.error('截图失败:', error)
    errorDialogRef.value?.showError('截图失败', '截图过程中发生错误', `错误详情：${error.message}`)
  }
}

// 获取图片缩放比例
const getImageScale = () => {
  if (!imageRef.value) return { x: 1, y: 1 }
  
  const img = imageRef.value
  if (img.naturalWidth && img.offsetWidth) {
    return {
      x: img.naturalWidth / img.offsetWidth,
      y: img.naturalHeight / img.offsetHeight
    }
  }
  return { x: 1, y: 1 }
}

// 开始选择区域
const startSelection = (event) => {
  if (!showAreaSelector.value) return
  
  // 阻止默认行为
  event.preventDefault()
  event.stopPropagation()
  
  const rect = event.target.getBoundingClientRect()
  
  // 使用响应式的缩放比例
  const x = (event.clientX - rect.left) * getImageScale().x
  const y = (event.clientY - rect.top) * getImageScale().y
  
  startPoint.value = { 
    x: Math.round(x), 
    y: Math.round(y) 
  }
  isSelecting.value = true
  
  selectedArea.value = {
    x: Math.round(x),
    y: Math.round(y),
    width: 0,
    height: 0
  }
}

// 更新选择区域
const updateSelection = (event) => {
  if (!isSelecting.value || !selectedArea.value) return
  
  // 阻止默认行为
  event.preventDefault()
  event.stopPropagation()
  
  const rect = event.target.getBoundingClientRect()
  
  // 使用响应式的缩放比例
  const x = (event.clientX - rect.left) * getImageScale().x
  const y = (event.clientY - rect.top) * getImageScale().y
  
  const startX = Math.min(startPoint.value.x, x)
  const startY = Math.min(startPoint.value.y, y)
  const width = Math.abs(x - startPoint.value.x)
  const height = Math.abs(y - startPoint.value.y)
  
  // 确保坐标值为整数，避免精度问题
  selectedArea.value = {
    x: Math.round(startX),
    y: Math.round(startY),
    width: Math.round(width),
    height: Math.round(height)
  }
}

// 结束选择区域
const endSelection = (event) => {
  if (event) {
    event.preventDefault()
    event.stopPropagation()
  }
  isSelecting.value = false
}

// 确认区域选择
const confirmAreaSelection = () => {
  console.log('确认区域选择被点击')
  console.log('selectedArea:', selectedArea.value)
  console.log('fullScreenshot:', fullScreenshot.value ? '存在' : '不存在')
  
  if (!selectedArea.value) {
    console.log('没有选择区域')
    return
  }
  
  try {
    // 更新区域选择器组件的截图区域
    if (areaSelectorRef.value) {
      console.log('更新区域选择器组件')
      Object.assign(areaSelectorRef.value.screenshotArea, selectedArea.value)
      areaSelectorRef.value.screenshotArea.image = fullScreenshot.value
      
      // 生成预览图片，传递选择的区域和截图数据
      if (areaSelectorRef.value.generatePreviewImage) {
        areaSelectorRef.value.generatePreviewImage(selectedArea.value, fullScreenshot.value)
        console.log('预览图片生成完成')
      } else {
        console.error('generatePreviewImage方法不存在')
      }
    } else {
      console.error('areaSelectorRef.value为空')
    }
    
    // 触发区域选择完成事件
    handleAreaSelected(selectedArea.value)
    console.log('区域选择完成事件已触发')
    
    // 关闭弹窗
    showAreaSelector.value = false
    console.log('弹窗已关闭')
  } catch (error) {
    console.error('确认区域选择时发生错误:', error)
    errorDialogRef.value?.showError('区域选择失败', '确认区域选择时发生错误', `错误详情：${error.message}`)
  }
}

// 取消区域选择
const cancelAreaSelection = () => {
  showAreaSelector.value = false
  selectedArea.value = null
}

// 处理搜索结果
const handleSearchResults = (results) => {
  answerDisplayRef.value?.updateSearchResults(results)
}

// 处理搜索错误
const handleSearchError = (error) => {
  console.error('搜索失败:', error)
  errorDialogRef.value?.showError('搜索失败', '搜索过程中发生错误', `错误详情：${error.message}`)
}

// 处理下一题错误
const handleNextQuestionError = (error) => {
  console.error('下一题失败:', error)
  errorDialogRef.value?.showError('下一题失败', '下一题过程中发生错误', `错误详情：${error.message}`)
}

// 处理截图更新
const handleUpdateScreenshot = (newScreenshot) => {
  if (areaSelectorRef.value) {
    areaSelectorRef.value.selectedAreaImage = newScreenshot
  }
}

// 获取OCR配置
const getOCRConfig = () => {
  return ocrConfigRef.value?.ocrConfig || {}
}

// 获取准确率筛选状态
const getAccuracyFilters = () => {
  return answerDisplayRef.value ? answerDisplayRef.value.accuracyFilters : accuracyFilters
}

// 初始化
onMounted(() => {
  // 答案页面默认为空，用户需要导入数据
  answerDisplayRef.value?.updateAnswers([])
})
</script>

<template>
  <div class="app-container">
    <div class="main-layout">
      <!-- 左侧配置区域 -->
      <div class="config-panel" :style="{ width: leftPanelWidth + 'px' }">
        <div class="config-content">
          <!-- OCR配置区域 -->
          <OCRConfig ref="ocrConfigRef" />
          
          <!-- 数据导入配置区域 -->
          <ImportConfig 
            ref="importConfigRef"
            @import-success="handleImportSuccess"
            @import-error="handleImportError"
          />
          
          <!-- 区域选择配置 -->
          <AreaSelector 
            ref="areaSelectorRef"
            @area-selected="handleAreaSelected"
            @start-area-selection="startAreaSelection"
          />
          
          <!-- 功能区 -->
          <FunctionArea 
            ref="functionAreaRef"
            :screenshot-area="areaSelectorRef ? areaSelectorRef.screenshotArea : {}"
            :answers="currentAnswers"
            :ocr-config="getOCRConfig()"
            :accuracy-filters="getAccuracyFilters()"
            @search-results="handleSearchResults"
            @search-error="handleSearchError"
            @update-screenshot="handleUpdateScreenshot"
            @next-question-error="handleNextQuestionError"
          />
        </div>
      </div>

      <!-- 可拖动的分隔条 -->
      <div 
        class="resizer" 
        @mousedown="startResize"
        @touchstart="startResize"
      ></div>

      <!-- 右侧答案显示区域 -->
      <div class="answer-panel">
        <AnswerDisplay ref="answerDisplayRef" />
      </div>
    </div>

    <!-- 错误弹窗 -->
    <ErrorDialog ref="errorDialogRef" />
    
    <!-- 全局区域选择弹窗 -->
    <t-dialog
      v-model:visible="showAreaSelector"
      title="划区截图 - 请在下方的截图中选择区域并点击确定"
      width="98%"
      height="98%"
      :close-on-overlay-click="false"
      :close-on-esc-keydown="false"
      :destroy-on-close="false"
      class="area-selector-dialog"
    >
      <div class="area-selector">
        <!-- 提示信息 -->
        <div class="area-tip">
          <t-icon name="info-circle" />
          <span>请在下方的截图中拖拽鼠标选择您需要的区域，选择完成后点击"确定"按钮</span>
        </div>
        
        <!-- 上方控制区域 -->
        <div class="area-controls-top">
          <div class="area-info">
            <span v-if="selectedArea" class="info-item">选择区域: {{ selectedArea.x }}, {{ selectedArea.y }} - {{ selectedArea.x + selectedArea.width }} x {{ selectedArea.y + selectedArea.height }}</span>
            <span v-if="selectedArea" class="info-item">缩放比例: {{ getImageScale().x.toFixed(2) }} x {{ getImageScale().y.toFixed(2) }}</span>
            <span v-if="selectedArea" class="info-item">显示区域: {{ Math.round(selectedArea.x / getImageScale().x) }}, {{ Math.round(selectedArea.y / getImageScale().y) }} - {{ Math.round(selectedArea.width / getImageScale().x) }} x {{ Math.round(selectedArea.height / getImageScale().y) }}</span>
          </div>
        </div>
        
        <!-- 截图显示区域 -->
        <div class="screenshot-container">
          <img 
            ref="imageRef"
            :src="fullScreenshot" 
            alt="全屏截图" 
            class="full-screenshot"
            style="width: 100% !important; height: 100% !important; object-fit: fill !important;"
            @mousedown="startSelection"
            @mousemove="updateSelection"
            @mouseup="endSelection"
            @mouseleave="endSelection"
            @dragstart.prevent
            @selectstart.prevent
            @load="imageRef = $event.target"
            draggable="false"
          />
          <div
            v-if="selectedArea && (selectedArea.width > 0 || selectedArea.height > 0)"
            class="selection-box"
            :style="{
              left: (selectedArea.x / getImageScale().x) + 'px',
              top: (selectedArea.y / getImageScale().y) + 'px',
              width: (selectedArea.width / getImageScale().x) + 'px',
              height: (selectedArea.height / getImageScale().y) + 'px'
            }"
          ></div>
        </div>
      </div>
      
      <template #footer>
        <div class="dialog-footer">
          <t-button @click="confirmAreaSelection" variant="base" :disabled="!selectedArea || (selectedArea.width === 0 && selectedArea.height === 0)">
            确定
          </t-button>
          <t-button @click="cancelAreaSelection" variant="base">
            取消
          </t-button>
        </div>
      </template>
    </t-dialog>
  </div>
</template>

<style scoped>
.app-container {
  height: 100vh;
  width: 100vw;
  overflow: hidden;
  background: #ffffff;
  position: relative;
}

.main-layout {
  display: flex;
  height: 100%;
  width: 100%;
  position: relative;
  z-index: 1;
}

.config-panel {
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(10px);
  border-right: 1px solid rgba(255, 255, 255, 0.2);
  display: flex;
  flex-direction: column;
  box-shadow: 2px 0 20px rgba(0, 0, 0, 0.1);
}

.config-content {
  padding: 20px;
  height: 100%;
  overflow-y: auto;
}

.answer-panel {
  background: rgba(255, 255, 255, 0.9);
  backdrop-filter: blur(10px);
  flex: 1;
  display: flex;
  flex-direction: column;
  box-shadow: -2px 0 20px rgba(0, 0, 0, 0.1);
}

.resizer {
  width: 6px;
  background: transparent;
  cursor: col-resize;
  position: relative;
  transition: all 0.2s ease;
  margin: 0 1px;
}

.resizer:hover {
  background: transparent;
}

.resizer:active {
  background: transparent;
}

.resizer::before {
  content: '';
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 1px;
  height: 20px;
  background: transparent;
  border-radius: 1px;
}

.resizer::after {
  content: '';
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 1px;
  height: 10px;
  background: transparent;
  border-radius: 1px;
}

/* 区域选择弹窗样式 */
.area-selector-dialog {
  z-index: 9999;
}

/* 确保弹框显示在屏幕最上方 */
.area-selector-dialog .t-dialog {
  top: 0 !important;
  margin-top: 0 !important;
  transform: none !important;
}

.area-selector {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.area-tip {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: rgba(0, 82, 217, 0.1);
  border: 1px solid rgba(0, 82, 217, 0.2);
  border-radius: 8px;
  margin-bottom: 12px;
  color: #0052d9;
  font-size: 14px;
  font-weight: 500;
}

.area-tip .t-icon {
  font-size: 16px;
  flex-shrink: 0;
}

.area-controls-top {
  padding: 10px;
  background: rgba(255, 255, 255, 0.9);
  border-bottom: 1px solid rgba(0, 0, 0, 0.1);
  flex-shrink: 0;
}

.area-info {
  display: flex;
  gap: 20px;
  flex-wrap: wrap;
}

.info-item {
  font-size: 12px;
  color: #666;
  background: rgba(0, 0, 0, 0.05);
  padding: 4px 8px;
  border-radius: 4px;
}

.screenshot-container {
  flex: 1;
  position: relative;
  overflow: hidden;
  background: #000;
}

.full-screenshot {
  display: block;
  max-width: 100%;
  max-height: 100%;
  cursor: crosshair;
}

.selection-box {
  position: absolute;
  border: 2px solid #0052d9;
  background: rgba(0, 82, 217, 0.1);
  pointer-events: none;
  z-index: 10;
}

.dialog-footer {
  display: flex;
  gap: 10px;
  justify-content: flex-end;
  padding: 10px;
}
</style>
