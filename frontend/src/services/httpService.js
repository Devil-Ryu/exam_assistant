// HTTP服务 - 处理与后端的HTTP通信

const API_BASE_URL = 'http://localhost:8088'

/**
 * 搜索答案
 * @param {string} query - 搜索查询
 * @param {Object} filters - 过滤条件
 * @returns {Promise<Array>} 搜索结果
 */
export async function searchAnswers(query, filters = {}) {
  try {
    const response = await fetch(`${API_BASE_URL}/api/search`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        query,
        filters
      })
    })

    if (!response.ok) {
      throw new Error(`HTTP请求失败: ${response.status} ${response.statusText}`)
    }

    const data = await response.json()
    
    if (!data.success) {
      throw new Error(data.message || '搜索失败')
    }

    return data.results || []
  } catch (error) {
    console.error('搜索答案失败:', error)
    throw error
  }
}

/**
 * 解析CSV文件
 * @param {string} filePath - 文件路径
 * @param {string} encoding - 文件编码
 * @param {string} optionSeparator - 选项分隔符
 * @param {string} answerSeparator - 答案分隔符
 * @returns {Promise<Array>} 解析结果
 */
export async function parseCSVFile(filePath, encoding, optionSeparator, answerSeparator) {
  try {
    const response = await fetch(`${API_BASE_URL}/api/parse-csv`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        filePath,
        encoding,
        optionSeparator,
        answerSeparator
      })
    })

    if (!response.ok) {
      throw new Error(`HTTP请求失败: ${response.status} ${response.statusText}`)
    }

    const data = await response.json()
    
    if (!data.success) {
      throw new Error(data.message || 'CSV解析失败')
    }

    return data.results || []
  } catch (error) {
    console.error('CSV解析失败:', error)
    throw error
  }
}

/**
 * 设置全局答案
 * @param {Array} answers - 答案数组
 * @returns {Promise<string>} 设置结果消息
 */
export async function setGlobalAnswers(answers) {
  try {
    const response = await fetch(`${API_BASE_URL}/api/set-global-answers`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        answers
      })
    })

    if (!response.ok) {
      throw new Error(`HTTP请求失败: ${response.status} ${response.statusText}`)
    }

    const data = await response.json()
    
    if (!data.success) {
      throw new Error(data.message || '设置全局答案失败')
    }

    return data.message || ''
  } catch (error) {
    console.error('设置全局答案失败:', error)
    throw error
  }
}

/**
 * 获取全局答案
 * @returns {Promise<Array>} 全局答案数组
 */
export async function getGlobalAnswers() {
  try {
    const response = await fetch(`${API_BASE_URL}/api/get-global-answers`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      }
    })

    if (!response.ok) {
      throw new Error(`HTTP请求失败: ${response.status} ${response.statusText}`)
    }

    const data = await response.json()
    
    if (!data.success) {
      throw new Error(data.message || '获取全局答案失败')
    }

    return data.answers || []
  } catch (error) {
    console.error('获取全局答案失败:', error)
    throw error
  }
}

/**
 * 测试OCR连接
 * @param {Object} config - OCR配置
 * @returns {Promise<string>} 测试结果
 */
export async function testOCRConnection(config) {
  try {
    const response = await fetch(`${API_BASE_URL}/api/test-ocr`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        config
      })
    })

    if (!response.ok) {
      throw new Error(`HTTP请求失败: ${response.status} ${response.statusText}`)
    }

    const data = await response.json()
    
    if (!data.success) {
      throw new Error(data.message || 'OCR测试失败')
    }

    return data.result || ''
  } catch (error) {
    console.error('OCR测试失败:', error)
    throw error
  }
}

/**
 * 截图
 * @returns {Promise<string>} 截图结果（base64图片数据）
 */
export async function takeScreenshot() {
  try {
    const response = await fetch(`${API_BASE_URL}/api/take-screenshot`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      }
    })

    if (!response.ok) {
      throw new Error(`HTTP请求失败: ${response.status} ${response.statusText}`)
    }

    const data = await response.json()
    
    if (!data.success) {
      throw new Error(data.message || '截图失败')
    }

    return data.image || ''
  } catch (error) {
    console.error('截图失败:', error)
    throw error
  }
}

/**
 * 执行OCR
 * @param {Object} area - 截图区域
 * @param {Object} config - OCR配置
 * @returns {Promise<string>} OCR识别结果
 */
export async function performOCR(area, config) {
  try {
    const response = await fetch(`${API_BASE_URL}/api/perform-ocr`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        area,
        config
      })
    })

    if (!response.ok) {
      throw new Error(`HTTP请求失败: ${response.status} ${response.statusText}`)
    }

    const data = await response.json()
    
    if (!data.success) {
      throw new Error(data.message || 'OCR执行失败')
    }

    return data.result || ''
  } catch (error) {
    console.error('OCR执行失败:', error)
    throw error
  }
}

/**
 * 测试HTTP连接
 * @returns {Promise<boolean>} 连接是否成功
 */
export async function testConnection() {
  try {
    const response = await fetch(`${API_BASE_URL}/api/search`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        query: 'test',
        filters: {}
      })
    })

    return response.ok
  } catch (error) {
    console.error('连接测试失败:', error)
    return false
  }
}
