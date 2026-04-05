package main

import (
	"fmt"

	"filetransfer/handler"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化上传目录
	if err := handler.InitUploadDir(); err != nil {
		fmt.Println("创建上传目录失败:", err)
		return
	}

	router := gin.Default()

	// 无需CORS配置，使用Vite代理解决跨域

	// 上传文件
	router.POST("/upload", handler.UploadFile)

	// 下载文件
	router.GET("/download/:filename", handler.DownloadFile)

	// 获取文件列表
	router.GET("/files", handler.GetFiles)

	// 预览文件
	router.GET("/preview/:filename", handler.PreviewFile)

	// 获取用户列表
	router.GET("/users", handler.GetUsers)

	// 群发文件
	router.POST("/group-send", handler.GroupSendFile)

	fmt.Println("服务器启动在 :8080")
	if err := router.Run(":8080"); err != nil {
		fmt.Println("服务器启动失败:", err)
	}
}
