<template>
  <div class="answer-content">
    <h2>答案列表</h2>
    
    <!-- 搜索结果 -->
    <div v-if="hasSearched && searchResults.length > 0" class="search-results">
      <div class="search-results-header">
        <h3>搜索结果 ({{ filteredSearchResults.length }}/{{ searchResults.length }}条)</h3>
        <div class="filter-controls" v-if="searchResults.length > 0">
          <t-select 
            v-model="selectedSearchTypeFilters" 
            placeholder="筛选题目类型（可多选）"
            multiple
            style="width: 300px;"
            :max="availableSearchTypes.length"
            @change="handleSearchFilterChange"
          >
            <t-option 
              v-for="type in availableSearchTypes" 
              :key="type" 
              :value="type" 
              :label="type" 
            />
          </t-select>
        </div>
      </div>
      <div class="search-stats">
        <div class="filter-tip">
          <t-icon name="info-circle" />
          <span>点击下方标签可过滤对应准确率的答案</span>
        </div>
      </div>
        <div class="accuracy-filters">
          <button 
            class="filter-button filter-high" 
            :class="{ 'filter-active': accuracyFilters.high }"
            @click="toggleAccuracyFilter('high')"
          >
            高准确率 (≥80%): {{ filteredSearchResults.filter(r => r.score >= 0.8).length }}个
          </button>
          <button 
            class="filter-button filter-medium" 
            :class="{ 'filter-active': accuracyFilters.medium }"
            @click="toggleAccuracyFilter('medium')"
          >
            中准确率 (50%-79%): {{ filteredSearchResults.filter(r => r.score >= 0.5 && r.score < 0.8).length }}个
          </button>
          <button 
            class="filter-button filter-low" 
            :class="{ 'filter-active': accuracyFilters.low }"
            @click="toggleAccuracyFilter('low')"
          >
            低准确率 (<50%): {{ filteredSearchResults.filter(r => r.score < 0.5).length }}个
          </button>
        </div>
        
      <div class="answer-cards">
        <t-card
          v-for="(result, index) in filteredSearchResults"
          :key="index"
          class="answer-card"
        >
          <template #header>
            <div class="card-header">
              <t-tag theme="primary">{{ result.item.type }}</t-tag>
            </div>
          </template>
          <div class="card-content">
            <div class="card-main">
              <div class="card-text">
                
                <p><strong>题目:</strong> <span v-html="highlightText(result.item.question, result.questionMatches)"></span></p>
                <div v-if="result.item.options.length > 0">
                  <p><strong>选项:</strong></p>
                  <ul>
                    <li v-for="option in result.item.options" :key="option" v-html="highlightText(option, result.optionMatches[option] || [])"></li>
                  </ul>
                </div>
                <p><strong>答案:</strong></p>
                <div class="answer-list">
                  <div v-for="(ans, index) in result.item.answer" :key="index" class="answer-item">
                    <span>{{ ans }}</span>
                  </div>
                </div>
                <p><strong>匹配到文本:</strong> {{  result.matched || '未匹配到文本' }}</p>
              </div>
              <div class="match-score-container" :style="getMatchScoreColor(result.score)">
                <div class="match-score-content">
                  <span class="match-score-label">准确率</span>
                  <span class="match-score-text">{{ (result.score * 100).toFixed(1) }}%</span>
                </div>
              </div>
            </div>
          </div>
        </t-card>
      </div>
    </div>

    <!-- 搜索结果为空状态 -->
    <div v-else-if="hasSearched && searchResults.length === 0" class="empty-state">
      <div class="empty-content">
        <div class="empty-icon">
          <svg width="64" height="64" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <circle cx="11" cy="11" r="8" stroke="currentColor" stroke-width="2"/>
            <path d="M21 21L16.65 16.65" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        </div>
        <h3>未找到匹配的答案</h3>
        <p>请尝试调整OCR识别文本或检查答案库是否包含相关内容</p>
      </div>
    </div>

    <!-- 所有答案 -->
    <div v-else class="all-answers">
      <div class="answers-header">
        <h3>所有答案 ({{ filteredAnswers.length }}/{{ answers.length }}条)</h3>
        <div class="filter-controls" v-if="answers.length > 0">
          <t-select 
            v-model="selectedTypeFilters" 
            placeholder="选择题目类型（可多选）"
            multiple
            style="width: 300px;"
            :max="availableTypes.length"
            @change="handleFilterChange"
          >
            <t-option 
              v-for="type in availableTypes" 
              :key="type" 
              :value="type" 
              :label="type" 
            />
          </t-select>
        </div>
      </div>
      <div v-if="answers.length > 0" class="answer-cards">
        <t-card
          v-for="(answer, index) in filteredAnswers"
          :key="index"
          class="answer-card"
        >
          <template #header>
            <div class="card-header">
              <t-tag theme="primary">{{ answer.type }}</t-tag>
            </div>
          </template>
          <div class="card-content">
            <p><strong>题目:</strong> {{ answer.question }}</p>
            <div v-if="answer.options.length > 0">
              <p><strong>选项:</strong></p>
              <ul>
                <li v-for="option in answer.options" :key="option">
                  {{ option }}
                </li>
              </ul>
            </div>
            <p><strong>答案:</strong></p>
            <div class="answer-list">
              <div v-for="(ans, index) in answer.answer" :key="index" class="answer-item">
                {{ ans }}
              </div>
            </div>
          </div>
        </t-card>
      </div>
      
      <!-- 题目类型统计 -->
      <div v-if="answers.length > 0" class="type-stats-inline">
        <span class="stats-label">题目类型统计:</span>
        <span 
          v-for="(count, type) in getFilteredTypeStats()" 
          :key="type" 
          class="type-stat-item-inline"
        >
          {{ type }}: {{ count }}题
        </span>
      </div>
    </div>
    
    <!-- 空状态 -->
    <div v-if="answers.length === 0" class="empty-state">
      <div class="empty-content">
        <div class="empty-icon">
          <svg width="64" height="64" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M12 2L2 7L12 12L22 7L12 2Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            <path d="M2 17L12 22L22 17" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            <path d="M2 12L12 17L22 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        </div>
        <h3>暂无答案数据</h3>
        <p>请点击左侧的"导入答案"按钮来加载答案数据</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed } from 'vue'

const answers = ref([])
const searchResults = ref([])
const hasSearched = ref(false)
const selectedTypeFilters = ref([])
const selectedSearchTypeFilters = ref([])

// 准确度筛选状态
const accuracyFilters = reactive({
  high: true,   // 高准确率 (≥80%)
  medium: true, // 中准确率 (50%-79%)
  low: true     // 低准确率 (<50%)
})

// 准确度筛选切换函数
const toggleAccuracyFilter = (type) => {
  accuracyFilters[type] = !accuracyFilters[type]
}

// 计算属性：筛选后的答案列表
const filteredAnswers = computed(() => {
  // 如果没有选择任何类型，显示所有答案
  if (selectedTypeFilters.value.length === 0) {
    return answers.value
  }
  // 否则只显示选中类型的答案
  return answers.value.filter(answer => selectedTypeFilters.value.includes(answer.type))
})

// 计算属性：可用的题目类型
const availableTypes = computed(() => {
  const types = new Set(answers.value.map(answer => answer.type))
  return Array.from(types).sort()
})

// 计算属性：搜索结果中可用的题目类型
const availableSearchTypes = computed(() => {
  const types = new Set(searchResults.value.map(result => result.item.type))
  return Array.from(types).sort()
})

// 计算属性：筛选后的搜索结果
const filteredSearchResults = computed(() => {
  let results = searchResults.value

  // 首先按题目类型筛选
  if (selectedSearchTypeFilters.value.length > 0) {
    results = results.filter(result => selectedSearchTypeFilters.value.includes(result.item.type))
  }

  // 然后按准确度筛选
  results = results.filter(result => {
    const score = result.score
    if (score >= 0.8) {
      return accuracyFilters.high
    } else if (score >= 0.5) {
      return accuracyFilters.medium
    } else {
      return accuracyFilters.low
    }
  })

  return results
})

// 处理筛选器变化
const handleFilterChange = (value) => {
  selectedTypeFilters.value = value
}

// 处理搜索结果筛选器变化
const handleSearchFilterChange = (value) => {
  selectedSearchTypeFilters.value = value
}

// 重置搜索状态
const resetSearchState = () => {
  searchResults.value = []
  hasSearched.value = false
  selectedSearchTypeFilters.value = []
  // 重置准确度筛选状态
  accuracyFilters.high = true
  accuracyFilters.medium = true
  accuracyFilters.low = true
}

// 获取匹配度颜色
const getMatchScoreColor = (score) => {
  if (score >= 0.8) {
    return { background: '#f6ffed', color: '#52c41a', border: '#b7eb8f' } // 绿色
  } else if (score >= 0.5) {
    return { background: '#fff7e6', color: '#fa8c16', border: '#ffd591' } // 橙色
  } else {
    return { background: '#fff2f0', color: '#ff4d4f', border: '#ffccc7' } // 红色
  }
}

// 获取筛选后的题目类型统计
const getFilteredTypeStats = () => {
  const stats = {}
  filteredAnswers.value.forEach(answer => {
    const type = answer.type
    stats[type] = (stats[type] || 0) + 1
  })
  return stats
}
// 高亮匹配的文本
const highlightText = (text, matches) => {
  if (!text || !matches || !Array.isArray(matches) || matches.length === 0) {
    return text
  }

  // 将文本转换为字符数组，支持中文
  const chars = Array.from(text)
  const highlightedChars = [...chars]
  
  // 对每个匹配位置进行高亮处理
  matches.forEach(position => {
    if (typeof position === 'number' && position >= 0 && position < chars.length) {
      // 在指定位置添加高亮标记
      highlightedChars[position] = `<span class="highlight-match">${chars[position]}</span>`
    }
  })

  return highlightedChars.join('')
}

// 更新答案数据
const updateAnswers = (newAnswers) => {
  answers.value = newAnswers
  resetSearchState()
}

// 更新搜索结果
const updateSearchResults = (results) => {
  console.log('更新搜索结果:', results)
  
  // 处理搜索结果
  if (results === null) {
    // 搜索框为空，显示所有答案
    console.log('搜索框为空，显示所有答案')
    hasSearched.value = false
    searchResults.value = []
  } else if (results && results.length > 0) {
    // 有搜索结果
    console.log('第一个搜索结果:', results[0])
    console.log('第一个结果的matched字段:', results[0].matched)
    hasSearched.value = true
    searchResults.value = results
  } else {
    // 搜索结果为空
    console.log('搜索结果为空')
    hasSearched.value = true
    searchResults.value = []
  }
}

// 暴露方法给父组件
defineExpose({
  updateAnswers,
  updateSearchResults,
  resetSearchState,
  accuracyFilters
})
</script>

<style scoped>
.answer-content {
  padding: 20px;
  height: 100%;
  overflow-y: auto;
  background: rgba(255, 255, 255, 0.7);
  backdrop-filter: blur(5px);
  border-radius: 8px;
  margin: 10px;
}

.answer-content h2 {
  margin: 0 0 20px 0;
  color: #2c3e50;
  font-weight: 700;
  font-size: 24px;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
}

.answer-content h3 {
  margin: 20px 0 15px 0;
  color: #34495e;
  font-weight: 600;
  font-size: 18px;
}

.answer-cards {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.answer-card {
  transition: all 0.3s ease;
  background: rgba(255, 255, 255, 0.9);
  backdrop-filter: blur(5px);
  border: 1px solid rgba(255, 255, 255, 0.3);
  box-shadow: 0 4px 15px rgba(0, 0, 0, 0.1);
}

.answer-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 25px rgba(0, 0, 0, 0.15);
  background: rgba(255, 255, 255, 1);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-content p {
  margin: 8px 0;
}

.card-content ul {
  margin: 8px 0;
}

.answer-list {
  margin: 8px 0;
}

.answer-item {
  padding: 8px 12px;
  margin: 4px 0;
  background-color: #f8f9fa;
  border-left: 3px solid #0052d9;
  border-radius: 4px;
  font-size: 14px;
  line-height: 1.4;
}

.answers-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  flex-wrap: wrap;
  gap: 16px;
}

.search-results-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  flex-wrap: wrap;
  gap: 16px;
}

.filter-controls {
  display: flex;
  align-items: center;
  gap: 12px;
}

.filter-controls .t-select {
  min-width: 300px;
}

.filter-controls .t-select__tags {
  max-width: 280px;
  overflow: hidden;
}

.card-content ul {
  margin: 8px 0;
  padding-left: 20px;
}

.card-content li {
  margin: 4px 0;
}

.search-results {
  margin-bottom: 30px;
}

.search-stats {
  display: flex;
  gap: 16px;
  margin-bottom: 10px;
  flex-wrap: wrap;
}

/* 准确度筛选按钮样式 */
.accuracy-filters {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.filter-button {
  padding: 6px 12px;
  border: 1px solid;
  border-radius: 16px;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  background: transparent;
  color: inherit;
  min-width: 80px;
  text-align: center;
}

.filter-button:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1);
}

/* 未激活状态 - 灰色 */
.filter-button:not(.filter-active) {
  border-color: #d9d9d9;
  color: #8c8c8c;
  background: #f5f5f5;
}

.filter-button:not(.filter-active):hover {
  border-color: #bfbfbf;
  color: #595959;
  background: #fafafa;
}

/* 激活状态 - 对应颜色 */
.filter-button.filter-high.filter-active {
  border-color: #52c41a;
  color: #52c41a;
}

.filter-button.filter-medium.filter-active {
  border-color: #faad14;
  color: #faad14;
}

.filter-button.filter-low.filter-active {
  border-color: #ff4d4f;
  color: #ff4d4f;
}

/* 筛选提示样式 */
.filter-tip {
  margin-top: 4px;
  padding: 6px 12px;
  background: rgba(24, 144, 255, 0.08);
  border: 1px solid rgba(24, 144, 255, 0.15);
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  font-size: 11px;
  color: #1890ff;
  width: 100%;
  opacity: 0.8;
}

.filter-tip .t-icon {
  font-size: 14px;
  color: #1890ff;
}

/* 内联题目类型统计样式 */
.type-stats-inline {
  margin-top: 16px;
  padding: 12px 16px;
  background: rgba(255, 255, 255, 0.8);
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.3);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.stats-label {
  font-size: 14px;
  font-weight: 600;
  color: #2c3e50;
  white-space: nowrap;
}

.type-stat-item-inline {
  padding: 4px 10px;
  background: rgba(255, 255, 255, 0.9);
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
  color: #495057;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  white-space: nowrap;
}

.empty-state {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 400px;
  padding: 40px;
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(10px);
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.3);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
}

.empty-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  max-width: 300px;
}

.empty-icon {
  font-size: 64px;
  margin-bottom: 16px;
  opacity: 0.6;
  color: #999;
}

.empty-icon svg {
  width: 64px;
  height: 64px;
  color: #999;
  opacity: 0.6;
}

.empty-content h3 {
  font-size: 16px;
  font-weight: 500;
  color: #333;
  margin: 0 0 8px 0;
  line-height: 1.4;
}

.empty-content p {
  font-size: 14px;
  color: #666;
  margin: 0;
  line-height: 1.5;
}

/* 卡片内容布局 */
.card-main {
  display: flex;
  gap: 16px;
  align-items: flex-start;
}

.card-text {
  flex: 1;
}

.match-score-container {
  min-width: 80px;
  height: 80px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2px solid;
  flex-shrink: 0;
}

.match-score-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
}

.match-score-label {
  font-size: 12px;
  font-weight: normal;
  margin-bottom: 4px;
  opacity: 0.8;
}

.match-score-text {
  font-size: 18px;
  font-weight: bold;
  text-align: center;
}

/* 高亮匹配文本样式 */
.highlight-match {
  background-color: rgba(255, 235, 59, 0.5) !important;
  color: #333 !important;
  padding: 1px 2px !important;
  border-radius: 2px !important;
  font-weight: 500 !important;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1) !important;
}

/* 全局样式，确保v-html中的样式生效 */
:deep(.highlight-match) {
  background-color: rgba(255, 235, 59, 0.5) !important;
  color: #333 !important;
  padding: 1px 2px !important;
  border-radius: 2px !important;
  font-weight: 500 !important;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1) !important;
}

@keyframes highlight-pulse {
  0% {
    background-color: #fff3cd;
    transform: scale(1);
  }
  50% {
    background-color: #ffeb3b;
    transform: scale(1.05);
  }
  100% {
    background-color: #ffeb3b;
    transform: scale(1);
  }
}
</style> 