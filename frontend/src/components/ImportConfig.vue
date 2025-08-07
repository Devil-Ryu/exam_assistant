<template>
  <div class="config-section">
    <h3>数据导入配置</h3>
    <div class="config-row">
      <div class="config-item">
        <label class="config-label">文件类型</label>
        <t-select v-model="importConfig.fileType" placeholder="选择文件类型" class="config-input">
          <t-option value="csv" label="CSV" />
        </t-select>
      </div>
      <div class="config-item">
        <label class="config-label">问题选项分隔符</label>
        <t-input
          v-model="importConfig.optionDelimiter"
          placeholder="如: \n 或 ,"
          class="config-input"
        />
      </div>
    </div>
    <div class="config-row">
      <div class="config-item">
        <label class="config-label">文件编码</label>
        <t-select v-model="importConfig.encoding" placeholder="选择文件编码" class="config-input">
          <t-option value="utf8" label="UTF-8" />
          <t-option value="gbk" label="GBK" />
        </t-select>
      </div>
      <div class="config-item">
        <label class="config-label">答案分隔符</label>
        <t-input
          v-model="importConfig.answerDelimiter"
          placeholder="如: , 或 \n"
          class="config-input"
        />
      </div>
    </div>
    <div class="config-row">
      <t-button @click="importAnswers" variant="base" class="config-button import-button" id="import-btn">
        导入答案
      </t-button>
    </div>
  </div>
</template>

<script setup>
import { reactive } from 'vue'
import { parseCSVFile, setGlobalAnswers } from '../services/httpService.js'

const importConfig = reactive({
  fileType: 'csv',
  encoding: 'utf8',
  answerDelimiter: '\\n',
  optionDelimiter: '\\n'
})

// 导入答案
const importAnswers = async () => {
  console.log('导入答案按钮被点击')
  
  try {
    // 使用Wails绑定文件对话框
    const { ExamService } = await import('../../bindings/changeme/index.js')
    
    // 根据文件类型设置对话框标题和文件类型
    const fileType = importConfig.fileType === 'csv' ? 'csv' : 'excel'
    const dialogTitle = importConfig.fileType === 'csv' ? '选择CSV答案文件' : '选择Excel答案文件'
    
    // 打开文件对话框
    const result = await ExamService.OpenFileDialog(dialogTitle, fileType)
    
    if (result.success && result.filePath) {
      console.log('选择的文件路径:', result.filePath)
      
      try {
        // 根据文件类型调用不同的导入方法
        console.log(`开始导入${importConfig.fileType.toUpperCase()}文件:`, result.filePath)
        let newAnswers
        
        if (importConfig.fileType === 'csv') {
          // 使用HTTP服务解析CSV文件
          newAnswers = await parseCSVFile(result.filePath, importConfig.encoding, importConfig.optionDelimiter, importConfig.answerDelimiter)
        } else {
          // TODO: 添加Excel文件导入支持
          throw new Error('Excel文件导入功能暂未实现')
        }
        
        // 验证解析结果
        if (!newAnswers || newAnswers.length === 0) {
          throw new Error('文件中没有找到有效的答案数据')
        }
        
        // 使用HTTP服务设置全局答案数据到后端
        await setGlobalAnswers(newAnswers)
        
        // 触发导入成功事件
        emit('import-success', newAnswers)
        console.log('导入成功，共导入', newAnswers.length, '条答案')
        
      } catch (error) {
        console.error('文件导入失败:', error)
        emit('import-error', error)
      }
    } else {
      console.log('用户取消了文件选择')
    }
  } catch (error) {
    console.error('导入失败:', error)
    emit('import-error', error)
  }
}

// 定义事件
const emit = defineEmits(['import-success', 'import-error'])

// 暴露配置给父组件
defineExpose({
  importConfig
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

.import-button {
  width: 100%;
}
</style> 