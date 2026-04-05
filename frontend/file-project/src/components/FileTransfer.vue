<template>
  <div class="container">
    <h1>文件传输系统</h1>
    
    <!-- 文件上传 -->
    <div class="upload-section">
      <h2>上传文件</h2>
      <form @submit.prevent="uploadFile">
        <input type="file" ref="fileInput" @change="handleFileChange" />
        <button type="submit" :disabled="!selectedFile">上传</button>
      </form>
      <div v-if="uploadProgress > 0 && uploadProgress < 100" class="progress">
        <div class="progress-bar" :style="{ width: uploadProgress + '%' }"></div>
        <span>{{ uploadProgress }}%</span>
      </div>
      <div v-if="uploadMessage" class="message" :class="uploadSuccess ? 'success' : 'error'">
        {{ uploadMessage }}
      </div>
    </div>
    
    <!-- 文件列表 -->
    <div class="file-list-section">
      <h2>文件列表</h2>
      <div v-if="!files || files.length === 0" class="empty">
        暂无文件
      </div>
      <table v-else class="file-table">
        <thead>
          <tr>
            <th>文件名</th>
            <th>类型</th>
            <th>大小</th>
            <th>修改时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="file in files" :key="file.name">
            <td>{{ file.name }}</td>
            <td>{{ file.type }}</td>
            <td>{{ formatSize(file.size) }}</td>
            <td>{{ formatTime(file.time) }}</td>
            <td>
              <button @click="previewFile(file)" v-if="file.type === 'text' || file.type === 'image' || file.type === 'video'">预览</button>
              <button @click="downloadFile(file.name)">下载</button>
              <button @click="openGroupSendModal(file)">群发</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    
    <!-- 预览模态框 -->
    <div v-if="previewVisible" class="modal-overlay" @click="closePreview">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h3>{{ previewFileInfo.name }}</h3>
          <button class="close-btn" @click="closePreview">&times;</button>
        </div>
        <div class="modal-body">
          <!-- 文本文件预览 -->
          <div v-if="previewFileInfo.type === 'text'" class="text-preview">
            <pre>{{ previewContent }}</pre>
          </div>
          <!-- 图片文件预览 -->
          <div v-else-if="previewFileInfo.type === 'image'" class="image-preview">
            <img :src="'/api/download/' + previewFileInfo.name" :alt="previewFileInfo.name" />
          </div>
          <!-- 视频文件预览 -->
          <div v-else-if="previewFileInfo.type === 'video'" class="video-preview">
            <video controls :src="'/api/download/' + previewFileInfo.name">
              您的浏览器不支持视频播放
            </video>
          </div>
        </div>
      </div>
    </div>
    
    <!-- 群发模态框 -->
    <div v-if="groupSendVisible" class="modal-overlay" @click="closeGroupSendModal">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h3>群发文件 - {{ groupSendFile.name }}</h3>
          <button class="close-btn" @click="closeGroupSendModal">&times;</button>
        </div>
        <div class="modal-body">
          <div class="group-send-section">
            <h4>选择接收用户</h4>
            <div class="user-list">
              <label v-for="user in users" :key="user.id" class="user-item">
                <input type="checkbox" v-model="selectedUsers" :value="user.id" />
                <span>{{ user.name }}</span>
              </label>
            </div>
            <div class="group-send-actions">
              <button @click="sendToAll">全选</button>
              <button @click="clearSelection">清空</button>
            </div>
            <div class="group-send-button">
              <button @click="groupSendFileAction" :disabled="selectedUsers.length === 0">确认群发</button>
            </div>
            <div v-if="groupSendMessage" class="message" :class="groupSendSuccess ? 'success' : 'error'">
              {{ groupSendMessage }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'

const fileInput = ref<HTMLInputElement>()
const selectedFile = ref<File | null>(null)
const uploadProgress = ref(0)
const uploadMessage = ref('')
const uploadSuccess = ref(false)
const files = ref<any[]>([])
const users = ref<any[]>([])

// 预览相关变量
const previewVisible = ref(false)
const previewFileInfo = ref({ name: '', type: '' })
const previewContent = ref('')

// 群发相关变量
const groupSendVisible = ref(false)
const groupSendFile = ref({ name: '' })
const selectedUsers = ref<string[]>([])
const groupSendMessage = ref('')
const groupSendSuccess = ref(false)

// 处理文件选择
const handleFileChange = (e: Event) => {
  const target = e.target as HTMLInputElement
  if (target.files && target.files.length > 0) {
    selectedFile.value = target.files[0]
  }
}

// 上传文件
const uploadFile = async () => {
  if (!selectedFile.value) return

  const formData = new FormData()
  formData.append('file', selectedFile.value)

  uploadProgress.value = 0
  uploadMessage.value = ''

  try {
    const response = await fetch('/api/upload', {
      method: 'POST',
      body: formData
    })

    if (response.ok) {
      const data = await response.json()
      uploadSuccess.value = true
      uploadMessage.value = data.message
      await fetchFiles()
      // 重置文件选择
      if (fileInput.value) {
        fileInput.value.value = ''
      }
      selectedFile.value = null
    } else {
      uploadSuccess.value = false
      uploadMessage.value = '上传失败'
    }
  } catch (error) {
    uploadSuccess.value = false
    uploadMessage.value = '上传失败: ' + error
  } finally {
    uploadProgress.value = 0
  }
}

// 下载文件
const downloadFile = (filename: string) => {
  window.open(`/api/download/${filename}`, '_blank')
}

// 预览文件
const previewFile = async (file: any) => {
  previewFileInfo.value = { name: file.name, type: file.type }
  
  if (file.type === 'text') {
    try {
      const response = await fetch(`/api/preview/${file.name}`)
      if (response.ok) {
        const data = await response.json()
        previewContent.value = data.content
      }
    } catch (error) {
      console.error('预览文件失败:', error)
      previewContent.value = '预览失败'
    }
  }
  
  previewVisible.value = true
}

// 关闭预览
const closePreview = () => {
  previewVisible.value = false
  previewContent.value = ''
}

// 打开群发模态框
const openGroupSendModal = async (file: any) => {
  groupSendFile.value = file
  selectedUsers.value = []
  groupSendMessage.value = ''
  
  // 获取用户列表
  await fetchUsers()
  
  groupSendVisible.value = true
}

// 关闭群发模态框
const closeGroupSendModal = () => {
  groupSendVisible.value = false
  groupSendMessage.value = ''
  selectedUsers.value = []
}

// 全选用户
const sendToAll = () => {
  selectedUsers.value = users.value.map(user => user.id)
}

// 清空选择
const clearSelection = () => {
  selectedUsers.value = []
}

// 执行群发
const groupSendFileAction = async () => {
  if (selectedUsers.value.length === 0) return
  
  try {
    const response = await fetch('/api/group-send', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        filename: groupSendFile.value.name,
        userIds: selectedUsers.value
      })
    })

    if (response.ok) {
      const data = await response.json()
      groupSendSuccess.value = true
      groupSendMessage.value = `文件已成功发送给: ${data.sentTo.join(', ')}`
    } else {
      groupSendSuccess.value = false
      groupSendMessage.value = '群发失败'
    }
  } catch (error) {
    groupSendSuccess.value = false
    groupSendMessage.value = '群发失败: ' + error
  }
}

// 获取文件列表
const fetchFiles = async () => {
  try {
    const response = await fetch('/api/files')
    if (response.ok) {
      files.value = await response.json()
    }
  } catch (error) {
    console.error('获取文件列表失败:', error)
  }
}

// 获取用户列表
const fetchUsers = async () => {
  try {
    const response = await fetch('/api/users')
    if (response.ok) {
      users.value = await response.json()
    }
  } catch (error) {
    console.error('获取用户列表失败:', error)
  }
}

// 格式化文件大小
const formatSize = (size: number): string => {
  if (size < 1024) {
    return size + ' B'
  } else if (size < 1024 * 1024) {
    return (size / 1024).toFixed(2) + ' KB'
  } else {
    return (size / (1024 * 1024)).toFixed(2) + ' MB'
  }
}

// 格式化时间
const formatTime = (time: string): string => {
  return new Date(time).toLocaleString()
}

// 组件挂载时获取文件列表
onMounted(() => {
  fetchFiles()
  fetchUsers()
})
</script>

<style scoped>
.container {
  max-width: 1000px;
  margin: 0 auto;
  padding: 20px;
}

h1 {
  text-align: center;
  color: #333;
  margin-bottom: 30px;
}

.upload-section, .file-list-section {
  margin: 20px 0;
  padding: 20px;
  border: 1px solid #ddd;
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

h2 {
  color: #555;
  margin-bottom: 15px;
  font-size: 18px;
}

form {
  display: flex;
  gap: 10px;
  align-items: center;
  flex-wrap: wrap;
}

input[type="file"] {
  flex: 1;
  min-width: 300px;
}

button {
  padding: 8px 16px;
  background-color: #4CAF50;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  transition: background-color 0.3s ease;
}

button:hover {
  background-color: #45a049;
}

button:disabled {
  background-color: #cccccc;
  cursor: not-allowed;
}

.progress {
  margin-top: 10px;
  height: 20px;
  background-color: #f0f0f0;
  border-radius: 10px;
  overflow: hidden;
  position: relative;
}

.progress-bar {
  height: 100%;
  background-color: #4CAF50;
  transition: width 0.3s ease;
}

.progress span {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  text-align: center;
  line-height: 20px;
  font-size: 12px;
  color: #333;
}

.message {
  margin-top: 10px;
  padding: 10px;
  border-radius: 4px;
}

.message.success {
  background-color: #d4edda;
  color: #155724;
  border: 1px solid #c3e6cb;
}

.message.error {
  background-color: #f8d7da;
  color: #721c24;
  border: 1px solid #f5c6cb;
}

.file-table {
  width: 100%;
  border-collapse: collapse;
  margin-top: 10px;
}

.file-table th, .file-table td {
  padding: 12px;
  text-align: left;
  border-bottom: 1px solid #ddd;
}

.file-table th {
  background-color: #f2f2f2;
  font-weight: bold;
  color: #333;
}

.file-table tr:hover {
  background-color: #f5f5f5;
}

.file-table td button {
  margin-right: 5px;
  font-size: 14px;
  padding: 6px 12px;
}

.empty {
  text-align: center;
  color: #999;
  padding: 40px;
  font-size: 16px;
}

/* 预览模态框样式 */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.modal-content {
  background-color: white;
  border-radius: 8px;
  width: 90%;
  max-width: 800px;
  max-height: 80vh;
  overflow: auto;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.modal-header {
  padding: 20px;
  border-bottom: 1px solid #ddd;
  display: flex;
  justify-content: space-between;
  align-items: center;
  background-color: #f9f9f9;
  border-radius: 8px 8px 0 0;
}

.modal-header h3 {
  margin: 0;
  color: #333;
}

.close-btn {
  background: none;
  border: none;
  font-size: 24px;
  cursor: pointer;
  color: #999;
  padding: 0;
  width: 30px;
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  transition: all 0.3s ease;
}

.close-btn:hover {
  background-color: #f0f0f0;
  color: #333;
}

.modal-body {
  padding: 20px;
}

/* 文本预览样式 */
.text-preview {
  max-height: 60vh;
  overflow: auto;
  background-color: #f8f8f8;
  padding: 15px;
  border-radius: 4px;
  border: 1px solid #ddd;
}

.text-preview pre {
  margin: 0;
  font-family: 'Courier New', Courier, monospace;
  font-size: 14px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-wrap: break-word;
}

/* 图片预览样式 */
.image-preview {
  text-align: center;
}

.image-preview img {
  max-width: 100%;
  max-height: 60vh;
  object-fit: contain;
  border-radius: 4px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

/* 视频预览样式 */
.video-preview {
  text-align: center;
}

.video-preview video {
  max-width: 100%;
  max-height: 60vh;
  border-radius: 4px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

/* 群发功能样式 */
.group-send-section {
  margin-top: 20px;
}

.group-send-section h4 {
  margin-bottom: 15px;
  color: #333;
}

.user-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 10px;
  margin-bottom: 20px;
  max-height: 300px;
  overflow-y: auto;
  padding: 10px;
  border: 1px solid #ddd;
  border-radius: 4px;
  background-color: #f9f9f9;
}

.user-item {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 8px;
  border-radius: 4px;
  transition: background-color 0.3s ease;
}

.user-item:hover {
  background-color: #f0f0f0;
}

.user-item input[type="checkbox"] {
  width: 16px;
  height: 16px;
  cursor: pointer;
}

.group-send-actions {
  display: flex;
  gap: 10px;
  margin-bottom: 20px;
}

.group-send-actions button {
  background-color: #6c757d;
}

.group-send-actions button:hover {
  background-color: #5a6268;
}

.group-send-button {
  text-align: center;
  margin-bottom: 20px;
}

.group-send-button button {
  background-color: #007bff;
  padding: 10px 20px;
  font-size: 16px;
}

.group-send-button button:hover {
  background-color: #0069d9;
}

.group-send-button button:disabled {
  background-color: #cccccc;
  cursor: not-allowed;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .container {
    padding: 10px;
  }
  
  .file-table {
    font-size: 14px;
  }
  
  .file-table th, .file-table td {
    padding: 8px;
  }
  
  .file-table td button {
    font-size: 12px;
    padding: 4px 8px;
    margin-right: 3px;
  }
  
  form {
    flex-direction: column;
    align-items: stretch;
  }
  
  input[type="file"] {
    min-width: auto;
  }
  
  .modal-content {
    width: 95%;
    max-height: 90vh;
  }
  
  .user-list {
    grid-template-columns: 1fr;
  }
  
  .group-send-actions {
    flex-direction: column;
  }
}
</style>