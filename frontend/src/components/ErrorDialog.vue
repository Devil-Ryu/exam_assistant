<template>
  <!-- 错误弹窗 -->
  <t-dialog
    v-model:visible="showErrorDialog"
    :title="errorDialogTitle"
    width="500px"
    :close-on-overlay-click="true"
    :close-on-esc-keydown="true"
  >
    <div class="error-dialog-content">
      <div class="error-message">
        {{ errorDialogContent }}
      </div>
      <div v-if="errorDialogDetails" class="error-details">
        <div class="details-title">详细信息：</div>
        <div class="details-content">
          {{ errorDialogDetails }}
        </div>
      </div>
    </div>
    
    <template #footer>
      <div class="dialog-footer">
        <t-button @click="closeErrorDialog" variant="base">
          确定
        </t-button>
      </div>
    </template>
  </t-dialog>
</template>

<script setup>
import { ref } from 'vue'

const showErrorDialog = ref(false)
const errorDialogTitle = ref('')
const errorDialogContent = ref('')
const errorDialogDetails = ref('')

// 显示错误弹窗
const showError = (title, content, details = '') => {
  errorDialogTitle.value = title
  errorDialogContent.value = content
  errorDialogDetails.value = details
  showErrorDialog.value = true
}

// 关闭错误弹窗
const closeErrorDialog = () => {
  showErrorDialog.value = false
}

// 暴露方法给父组件
defineExpose({
  showError
})
</script>

<style scoped>
.error-dialog-content {
  padding: 20px 0;
}

.error-message {
  font-size: 14px;
  color: #333;
  margin-bottom: 15px;
  line-height: 1.5;
}

.error-details {
  background: #f8f9fa;
  border: 1px solid #e9ecef;
  border-radius: 6px;
  padding: 15px;
  margin-top: 15px;
}

.details-title {
  font-weight: 600;
  color: #495057;
  margin-bottom: 8px;
  font-size: 13px;
}

.details-content {
  font-size: 12px;
  color: #6c757d;
  line-height: 1.6;
  white-space: pre-line;
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