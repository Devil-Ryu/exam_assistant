<template>
  <div class="config-section">
    <h3>区域选择配置</h3>
    <div class="area-controls">
      <t-button @click="selectArea" variant="base" class="config-button">
        点击选择区域
      </t-button>
      <t-tag>截图区域: {{ Math.round(screenshotArea.x) }}, {{ Math.round(screenshotArea.y) }}, {{ Math.round(screenshotArea.width) }}x{{ Math.round(screenshotArea.height) }}</t-tag>
    </div>
    <div class="screenshot-preview">
      <div class="preview-images">
        <img v-if="selectedAreaImage" :src="selectedAreaImage" alt="选择区域" class="preview-image" />
        <div v-else class="screenshot-placeholder">
          <span>暂无截图预览</span>
        </div>
      </div>

    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'

const screenshotArea = reactive({
  x: 0,
  y: 0,
  width: 0,
  height: 0,
  image: ''
})

const selectedAreaImage = ref('')
const showAreaSelector = ref(false)
const fullScreenshot = ref('')
const selectedArea = ref(null)
const isSelecting = ref(false)
const startPoint = ref({ x: 0, y: 0 })
const imageRef = ref(null)

// 定义事件
const emit = defineEmits(['area-selected', 'start-area-selection'])

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

// 选择区域
const selectArea = async () => {
  // 触发事件通知主应用开始区域选择
  emit('start-area-selection')
}

// 确认区域选择
const confirmAreaSelection = () => {
  if (!selectedArea.value) return
  
  // 更新截图区域
  Object.assign(screenshotArea, selectedArea.value)
  screenshotArea.image = fullScreenshot.value
  
  // 生成预览图片
  generatePreviewImage()
  
  // 触发区域选择完成事件
  emit('area-selected', screenshotArea)
  
  // 关闭弹窗
  showAreaSelector.value = false
}

// 取消区域选择
const cancelAreaSelection = () => {
  showAreaSelector.value = false
  selectedArea.value = null
}

// 生成预览图片
const generatePreviewImage = (area = null, screenshotData = null) => {
  const targetArea = area || selectedArea.value
  const targetScreenshot = screenshotData || fullScreenshot.value
  
  if (!targetScreenshot || !targetArea) {
    console.log('缺少截图数据或区域信息:', { targetScreenshot: !!targetScreenshot, targetArea: !!targetArea })
    return
  }
  
  const img = new Image()
  img.onload = () => {
    const canvas = document.createElement('canvas')
    const ctx = canvas.getContext('2d')
    
    // 设置预览尺寸
    const maxWidth = 200
    const maxHeight = 150
    
    // 计算缩放比例
    const scaleX = maxWidth / targetArea.width
    const scaleY = maxHeight / targetArea.height
    const scale = Math.min(scaleX, scaleY)
    
    const previewWidth = targetArea.width * scale
    const previewHeight = targetArea.height * scale
    
    canvas.width = previewWidth
    canvas.height = previewHeight
    
    // 绘制裁剪的区域
    ctx.drawImage(
      img,
      targetArea.x, targetArea.y, targetArea.width, targetArea.height,
      0, 0, previewWidth, previewHeight
    )
    
    selectedAreaImage.value = canvas.toDataURL('image/png')
    console.log('预览图片生成完成:', selectedAreaImage.value.substring(0, 50) + '...')
  }
  img.onerror = (error) => {
    console.error('图片加载失败:', error)
  }
  img.src = targetScreenshot
}

// 暴露配置给父组件
defineExpose({
  screenshotArea,
  selectedAreaImage,
  generatePreviewImage
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

.area-controls {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 10px;
  margin-bottom: 10px;
}

.area-controls .config-button {
  margin: 0;
  flex-shrink: 0;
  height: 24px;
  padding: 0 8px;
  font-size: 11px;
}

.area-controls .t-tag {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.screenshot-preview {
  margin-top: 10px;
  border: 1px solid #ddd;
  border-radius: 4px;
  overflow: hidden;
  height: 160px;
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f8f8f8;
}

.preview-images {
  display: flex;
  gap: 10px;
  width: 100%;
  height: 100%;
}

.preview-image {
  width: 100%;
  height: 100%;
  border-radius: 4px;
  border: 1px solid #e0e0e0;
}

.screenshot-placeholder {
  display: flex;
  justify-content: center;
  align-items: center;
  width: 100%;
  height: 100%;
  background-color: #f0f0f0;
  border-radius: 4px;
  border: 1px dashed #ccc;
}

.screenshot-placeholder span {
  color: #999;
  font-size: 14px;
}

.area-selector {
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 100%;
  max-height: 100%;
  overflow: hidden;
}

.area-controls-top {
  display: flex;
  justify-content: flex-start;
  align-items: center;
  padding: 8px 0;
  flex-shrink: 0;
}

.area-info {
  margin: 0;
  font-size: 12px;
  color: #666;
  flex: 1;
  line-height: 1.4;
  display: flex;
  gap: 20px;
  align-items: center;
}

.info-item {
  white-space: nowrap;
  padding: 6px 12px;
  background: rgba(255, 255, 255, 0.9);
  backdrop-filter: blur(5px);
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.3);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  font-weight: 500;
  color: #2c3e50;
}

.screenshot-container {
  position: relative;
  flex: 1;
  overflow: hidden;
  background: #f0f0f0;
  border-radius: 8px;
  margin-bottom: 16px;
  height: calc(100vh - 200px);
  min-height: 500px;
  display: flex;
  align-items: stretch;
  justify-content: stretch;
  width: 100%;
}

.full-screenshot {
  width: 100% !important;
  height: 100% !important;
  cursor: crosshair;
  user-select: none;
  -webkit-user-select: none;
  -moz-user-select: none;
  -ms-user-select: none;
  pointer-events: auto;
  -webkit-user-drag: none;
  -khtml-user-drag: none;
  -moz-user-drag: none;
  -o-user-drag: none;
  user-drag: none;
  object-fit: fill !important;
  border-radius: 8px;
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  margin: 0;
  padding: 0;
}

.selection-box {
  position: absolute;
  border: 2px solid #1890ff;
  background: rgba(24, 144, 255, 0.1);
  pointer-events: none;
  z-index: 10;
  border-radius: 4px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 0;
}

.dialog-footer .t-button {
  min-width: 80px;
}
</style> 