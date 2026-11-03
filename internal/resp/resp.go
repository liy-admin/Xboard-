package resp

import "github.com/gin-gonic/gin"

// 与原 ApiResponse trait 输出结构一致，前端无需改动。
func OK(c *gin.Context, data any) {
	c.JSON(200, gin.H{"status": "success", "message": "success", "data": data, "error": nil})
}

func Fail(c *gin.Context, code int, msg string) {
	c.AbortWithStatusJSON(code, gin.H{"status": "fail", "message": msg, "data": nil, "error": nil})
}
