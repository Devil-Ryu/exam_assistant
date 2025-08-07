<template>
  <div class="config-section">
    <h3>OCR配置</h3>
    <div class="config-row">
      <div class="config-item">
        <label class="config-label">OCR服务基础URL</label>
        <t-input
          v-model="ocrConfig.url"
          placeholder="请输入OCR服务基础URL，如: http://127.0.0.1:8080"
          class="config-input"
        />
      </div>
    </div>
    <div class="config-row">
      <t-button @click="testConnection" theme="primary" variant="base" class="config-button">
        测试连接
      </t-button>
      <div class="connection-status-right">
        <t-tag 
          :class="getConnectionStatusClass()"
          class="status-tag"
        >
          {{ ocrConfig.status || '未连接' }}
        </t-tag>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive } from 'vue'
import { testOCRConnection } from '../services/httpService.js'

const ocrConfig = reactive({
  mode: 'local',
  url: 'http://127.0.0.1:8080',
  apiKey: '',
  status: '未链接'
})

// 测试本地OCR连接
const testConnection = async () => {
  try {
    console.log('开始测试本地OCR连接')
    ocrConfig.status = '连接中'
    
    // 检查URL是否为空
    if (!ocrConfig.url || ocrConfig.url.trim() === '') {
      ocrConfig.status = '连接失败'
      throw new Error('OCR服务URL不能为空，请先设置服务地址')
    }
    
    // 使用HTTP服务测试OCR连接
    const result = await testOCRConnection(ocrConfig)
    
    if (result === '连接成功') {
      ocrConfig.status = '连接成功'
      console.log('OCR服务连接测试成功')
    } else {
      ocrConfig.status = '连接失败'
      console.error('OCR服务连接测试失败:', result)
    }
  } catch (error) {
    console.error('本地OCR连接测试失败:', error)
    ocrConfig.status = '连接失败'
  }
}

// 获取连接状态样式类
const getConnectionStatusClass = () => {
  if (ocrConfig.status === '连接成功') {
    return 'status-success'
  } else if (ocrConfig.status === '连接失败') {
    return 'status-error'
  } else if (ocrConfig.status === '连接中') {
    return 'status-connecting'
  } else {
    return 'status-default'
  }
}

// 暴露配置给父组件
defineExpose({
  ocrConfig
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

.config-row {
  display: flex;
  gap: 2px;
  margin-bottom: 2px;
  min-width: 0;
  align-items: center;
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

.config-button {
  height: 28px;
  font-size: 12px;
  margin-top: 4px;
}

.connection-status-right {
  display: flex;
  align-items: center;
  height: 100%;
  flex-shrink: 0;
  margin-left: auto;
}

.status-tag {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 12px;
  font-weight: 500;
  border: none;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 20px;
}

.status-success {
  background: linear-gradient(135deg, #a8e6cf, #88d8c0);
  color: #2d5a3d;
}

.status-error {
  background: linear-gradient(135deg, #ffb3ba, #ff8a95);
  color: #8b2635;
}

.status-connecting {
  background: linear-gradient(135deg, #fff3cd, #ffeaa7);
  color: #856404;
}

.status-default {
  background: linear-gradient(135deg, #e8f4fd, #d1e7dd);
  color: #6c757d;
}
</style> 