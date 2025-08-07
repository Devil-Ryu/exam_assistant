<template>
  <div class="config-section">
    <h3>功能区</h3>
    <div class="config-item">
      <label class="config-label">搜索内容</label>
      <t-textarea
        v-model="ocrResult"
        placeholder="可手动输入搜索内容，或点击下一题进行OCR识别自动搜索"
        :rows="4"
        class="config-input t-textarea"
      />
    </div>
    <div class="action-buttons">
      <t-button @click="searchAnswers" variant="base" class="action-button">
        搜索
      </t-button>
      <t-button @click="nextQuestion" variant="base" class="action-button">
        下一题
      </t-button>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { takeScreenshot, performOCR, searchAnswers as httpSearchAnswers } from '../services/httpService.js'

const props = defineProps({
  screenshotArea: {
    type: Object,
    required: true
  },
  ocrConfig: {
    type: Object,
    required: true
  },
  accuracyFilters: {
    type: Object,
    default: () => ({
      high: true,
      medium: true,
      low: true
    })
  }
})

const emit = defineEmits(['update-screenshot', 'next-question-error', 'search-results', 'search-error'])

const ocrResult = ref('')

// 下一题功能
const nextQuestion = async () => {
  try {
    console.log('开始下一题功能')
    
    // 检查是否有截图区域
    if (!props.screenshotArea || !props.screenshotArea.width || !props.screenshotArea.height) {
      alert('请先选择截图区域')
      return
    }
    
    // 1. 重新截取整个屏幕
    console.log('重新截取屏幕')
    const newScreenshot = await takeScreenshot()
    
    // 2. 从新截图中提取选择区域
    console.log('从新截图中提取选择区域')
    const newAreaImage = await cropImageForDisplay(newScreenshot, props.screenshotArea)
    
    // 3. 更新主页上的截图
    emit('update-screenshot', newAreaImage)
    console.log('已更新主页截图')
    
    // 4. 进行OCR识别
    console.log('开始OCR识别')
    const ocrText = await performOCRWithBackend(newScreenshot, props.screenshotArea)
    ocrResult.value = ocrText
    console.log('OCR识别结果:', ocrText)
    
    // 5. 自动进行搜索
    console.log('开始自动搜索')
    await searchAnswers()
    
    console.log('下一题完成')
  } catch (error) {
    console.error('下一题失败:', error)
    emit('next-question-error', error)
  }
}

// 搜索答案功能
const searchAnswers = async () => {
  try {
    console.log('搜索按钮被点击')
    console.log('OCR结果:', ocrResult.value)
    
    // 如果搜索内容为空，发送特殊信号表示显示所有答案
    if (!ocrResult.value.trim()) {
      console.log('搜索内容为空，显示所有答案')
      emit('search-results', null) // 使用null表示显示所有答案
      return
    }
    
    console.log('开始搜索:', ocrResult.value)
    
    // 构建筛选条件
    const filters = {
      accuracyFilters: props.accuracyFilters
    }
    
    // 调用HTTP搜索接口
    const results = await httpSearchAnswers(ocrResult.value, filters)
    console.log('HTTP接口返回结果:', results)
    
    // 显示所有匹配结果，按匹配度排序
    const sortedResults = results.sort((a, b) => b.score - a.score)
    
    console.log('搜索完成，找到', results.length, '条结果')
    
    // 触发搜索结果事件
    emit('search-results', sortedResults)
    
  } catch (error) {
    console.error('搜索失败:', error)
    emit('search-error', error)
  }
}

// 通过HTTP服务执行OCR识别
const performOCRWithBackend = async (screenshotData, area) => {
  try {
    console.log('开始通过HTTP服务执行OCR识别')
    console.log('使用OCR配置:', props.ocrConfig)
    
    // 检查OCR配置
    if (!props.ocrConfig || !props.ocrConfig.url) {
      throw new Error('OCR服务未配置，请先在OCR配置中设置服务URL')
    }
    
    // 创建截图区域对象
    const screenshotArea = {
      x: area.x,
      y: area.y,
      width: area.width,
      height: area.height,
      image: screenshotData
    }
    
    // 调用HTTP服务OCR识别，传入OCR配置
    const result = await performOCR(screenshotArea, props.ocrConfig)
    
    if (result && result.trim()) {
      console.log('OCR识别成功:', result)
      return result
    } else {
      console.log('OCR识别返回空结果')
      return "OCR识别未返回有效内容"
    }
  } catch (error) {
    console.error('OCR识别失败:', error)
    return "OCR识别失败: " + error.message
  }
}

// 裁剪图片用于主页显示
const cropImageForDisplay = (imageSrc, area) => {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.onload = () => {
      try {
        // 创建canvas
        const canvas = document.createElement('canvas')
        const ctx = canvas.getContext('2d')
        
        // 计算保持宽高比的显示尺寸
        const maxDisplayWidth = 300  // 最大显示宽度
        const maxDisplayHeight = 200 // 最大显示高度
        
        // 计算原始区域的宽高比
        const originalRatio = area.width / area.height
        const displayRatio = maxDisplayWidth / maxDisplayHeight
        
        let displayWidth, displayHeight
        
        if (originalRatio > displayRatio) {
          // 原始区域更宽，以宽度为准
          displayWidth = maxDisplayWidth
          displayHeight = maxDisplayWidth / originalRatio
        } else {
          // 原始区域更高，以高度为准
          displayHeight = maxDisplayHeight
          displayWidth = maxDisplayHeight * originalRatio
        }
        
        // 设置canvas尺寸
        canvas.width = displayWidth
        canvas.height = displayHeight
        
        // 绘制裁剪后的图片
        ctx.drawImage(
          img,
          area.x, area.y, area.width, area.height,  // 源图片裁剪区域
          0, 0, displayWidth, displayHeight          // 目标canvas区域
        )
        
        // 转换为base64
        const croppedImage = canvas.toDataURL('image/png')
        resolve(croppedImage)
      } catch (error) {
        reject(error)
      }
    }
    
    img.onerror = () => {
      reject(new Error('图片加载失败'))
    }
    
    img.src = imageSrc
  })
}

defineExpose({
  nextQuestion,
  searchAnswers,
  ocrResult
})
</script>

<style scoped>
.config-section {
  margin-top: 10px;
  padding: 12px;
  background: rgba(255, 255, 255, 0.8);
  border-radius: 12px;
  box-shadow: 0 4px 15px rgba(0, 0, 0, 0.1);
  min-width: 0;
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.3);
  backdrop-filter: blur(5px);
}

.config-section h3 {
  margin: 0 0 2px 0;
  color: #333;
  font-size: 14px;
  font-weight: 600;
}

.config-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
}

.config-label {
  font-size: 11px;
  color: #666;
  margin-bottom: 3px;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.config-input {
  margin-bottom: 4px;
  height: 28px;
  font-size: 12px;
  min-width: 0;
  width: 100%;
}

/* 特殊处理textarea */
.config-input.t-textarea {
  height: auto;
  min-height: 80px;
}

.action-buttons {
  display: flex;
  gap: 8px;
  margin-top: 4px;
}

.action-button {
  flex: 1;
}
</style> 