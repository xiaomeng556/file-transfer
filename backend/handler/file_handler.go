package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

var (
	uploadDir = "./uploads"
	wg        sync.WaitGroup
)

// 初始化上传目录
func InitUploadDir() error {
	return os.MkdirAll(uploadDir, 0755)
}

// UploadFile 处理文件上传（保留原接口，用于兼容旧逻辑）
func UploadFile(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "获取文件失败"})
		return
	}
	defer file.Close()

	// 上传目录
	uploadDir := "./uploads"
	// 自动创建目录
	_ = os.MkdirAll(uploadDir, 0755)

	// ✅ 核心：获取不重复文件名（自动加数字）
	savePath := getUniqueFileName(uploadDir, header.Filename)

	// 创建文件
	dst, err := os.Create(savePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建文件失败"})
		return
	}
	defer dst.Close()

	// 单线程写入文件，确保文件完整性
	bufferSize := 1024 * 1024 // 1MB
	buffer := make([]byte, bufferSize)

	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
			return
		}
		if n == 0 {
			break
		}

		if _, err := dst.Write(buffer[:n]); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "写入文件失败"})
			return
		}
	}

	// 获取保存后的文件名（用于返回给前端）
	saveName := filepath.Base(savePath)

	c.JSON(http.StatusOK, gin.H{
		"message":  "文件上传成功",
		"filename": header.Filename, // 原始文件名
		"saveName": saveName,        // 实际保存的文件名（重点）
		"size":     header.Size,
	})
}

// UploadChunk 处理文件块上传
func UploadChunk(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "获取文件失败"})
		return
	}
	defer file.Close()

	fileHash := c.PostForm("fileHash")
	chunkIndex := c.PostForm("chunkIndex")

	// 临时存储目录
	tempDir := filepath.Join("./temp", fileHash)
	_ = os.MkdirAll(tempDir, 0755)

	// 保存文件块
	chunkPath := filepath.Join(tempDir, chunkIndex)
	dst, err := os.Create(chunkPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建文件块失败"})
		return
	}
	defer dst.Close()

	// 写入文件块
	bufferSize := 1024 * 1024
	buffer := make([]byte, bufferSize)
	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
			return
		}
		if n == 0 {
			break
		}
		if _, err := dst.Write(buffer[:n]); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "写入文件失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "文件块上传成功"})
}

// MergeChunks 合并文件块
func MergeChunks(c *gin.Context) {
	var req struct {
		FileHash    string `json:"fileHash"`
		FileName    string `json:"fileName"`
		TotalChunks int    `json:"totalChunks"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	tempDir := filepath.Join("./temp", req.FileHash)
	uploadDir := "./uploads"
	_ = os.MkdirAll(uploadDir, 0755)

	// 创建最终文件
	savePath := getUniqueFileName(uploadDir, req.FileName)
	dst, err := os.Create(savePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建文件失败"})
		return
	}
	defer dst.Close()

	// 按顺序合并文件块
	for i := 0; i < req.TotalChunks; i++ {
		chunkPath := filepath.Join(tempDir, strconv.Itoa(i))
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件块失败"})
			return
		}

		// 写入文件块
		bufferSize := 1024 * 1024
		buffer := make([]byte, bufferSize)
		for {
			n, err := chunkFile.Read(buffer)
			if err != nil && err != io.EOF {
				chunkFile.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件块失败"})
				return
			}
			if n == 0 {
				break
			}
			if _, err := dst.Write(buffer[:n]); err != nil {
				chunkFile.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "写入文件失败"})
				return
			}
		}
		chunkFile.Close()
	}

	// 删除临时文件
	_ = os.RemoveAll(tempDir)

	// 获取保存后的文件名
	saveName := filepath.Base(savePath)

	c.JSON(http.StatusOK, gin.H{
		"message":  "文件上传成功",
		"filename": req.FileName,
		"saveName": saveName,
	})
}

// 获取不重复的文件名，格式：name.jpg → name(1).jpg → name(2).jpg
func getUniqueFileName(uploadDir, filename string) string {
	// 分离文件名和后缀
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)

	// 初始路径
	savePath := filepath.Join(uploadDir, filename)
	counter := 1

	// 如果文件存在，就循环尝试 (1)(2)...
	for {
		if _, err := os.Stat(savePath); os.IsNotExist(err) {
			break // 不存在，可用
		}
		// 文件已存在，重命名
		savePath = filepath.Join(uploadDir, fmt.Sprintf("%s(%d)%s", name, counter, ext))
		counter++
	}

	return savePath
}

// DownloadFile 处理文件下载
func DownloadFile(c *gin.Context) {
	filename := c.Param("filename")
	filePath := filepath.Join(uploadDir, filename)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "打开文件失败"})
		return
	}
	defer file.Close()

	// 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取文件信息失败"})
		return
	}

	// 设置响应头
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))
	c.Header("Cache-Control", "no-cache")

	// 多线程读取文件
	bufferSize := 1024 * 1024 // 1MB
	buffer := make([]byte, bufferSize)

	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
			return
		}
		if n == 0 {
			break
		}

		if _, err := c.Writer.Write(buffer[:n]); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "写入响应失败"})
			return
		}
	}

	c.Writer.Flush()
}

// GetFiles 获取文件列表
func GetFiles(c *gin.Context) {
	var files []gin.H

	// 读取上传目录
	dir, err := os.Open(uploadDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "打开上传目录失败"})
		return
	}
	defer dir.Close()

	fileInfos, err := dir.Readdir(-1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取目录失败"})
		return
	}

	for _, fileInfo := range fileInfos {
		if !fileInfo.IsDir() {
			files = append(files, gin.H{
				"name": fileInfo.Name(),
				"size": fileInfo.Size(),
				"time": fileInfo.ModTime(),
				"type": getFileType(fileInfo.Name()),
			})
		}
	}

	c.JSON(http.StatusOK, files)
}

// PreviewFile 预览文件内容
func PreviewFile(c *gin.Context) {
	filename := c.Param("filename")
	filePath := filepath.Join(uploadDir, filename)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	// 检查文件类型
	fileType := getFileType(filename)
	if fileType != "text" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持文本文件预览"})
		return
	}

	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"content":  string(content),
		"filename": filename,
	})
}

// getFileType 根据文件名判断文件类型
func getFileType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	// 文本文件类型
	textExts := []string{".txt", ".md", ".html", ".css", ".js", ".go", ".java", ".py", ".c", ".cpp", ".h"}
	for _, e := range textExts {
		if e == ext {
			return "text"
		}
	}

	// 图片文件类型
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp"}
	for _, e := range imageExts {
		if e == ext {
			return "image"
		}
	}

	// 视频文件类型
	videoExts := []string{".mp4", ".avi", ".mov", ".wmv", ".flv", ".mkv"}
	for _, e := range videoExts {
		if e == ext {
			return "video"
		}
	}

	return "other"
}

// 模拟用户数据
var users = []gin.H{
	{"id": "1", "name": "用户1"},
	{"id": "2", "name": "用户2"},
	{"id": "3", "name": "用户3"},
	{"id": "4", "name": "用户4"},
	{"id": "5", "name": "用户5"},
}

// GetUsers 获取用户列表
func GetUsers(c *gin.Context) {
	c.JSON(http.StatusOK, users)
}

// GroupSendFile 群发文件
func GroupSendFile(c *gin.Context) {
	var request struct {
		Filename string   `json:"filename"`
		UserIDs  []string `json:"userIds"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 检查文件是否存在
	filePath := filepath.Join(uploadDir, request.Filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	// 模拟群发操作
	// 在实际应用中，这里应该实现向指定用户发送文件的逻辑
	// 例如：将文件复制到用户目录、发送通知等

	// 记录群发信息
	sentUsers := []string{}
	for _, userID := range request.UserIDs {
		for _, user := range users {
			if user["id"] == userID {
				sentUsers = append(sentUsers, user["name"].(string))
				break
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "文件群发成功",
		"filename": request.Filename,
		"sentTo":   sentUsers,
	})
}

// DeleteFile 删除文件
func DeleteFile(c *gin.Context) {
	filename := c.Param("filename")
	filePath := filepath.Join(uploadDir, filename)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	// 删除文件
	if err := os.Remove(filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除文件失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "文件删除成功"})
}
