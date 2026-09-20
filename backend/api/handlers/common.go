// Package handlers 公共辅助函数：统一请求参数解析与错误响应，
// 避免各 handler 重复实现且忽略解析错误。
package handlers

import (
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// parseUintParam 解析路径参数为 uint。
// 解析失败或为 0 时直接写入 400 响应并返回 ok=false，调用方应立即 return。
//
// 说明：此前各 handler 普遍使用 `id, _ := strconv.ParseUint(...)` 忽略错误，
// 非法 id 会被静默转为 0，导致 Delete(&Model{}, 0)、Stop(0) 等语义不明的操作。
func parseUintParam(c *gin.Context, name string) (uint, bool) {
	raw := c.Param(name)
	v, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || v == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数 " + name + " 非法: " + raw,
		})
		return 0, false
	}
	return uint(v), true
}

// validatePort 校验端口号范围（1-65535），非法时写入 400 响应并返回 false。
func validatePort(c *gin.Context, field string, port int) bool {
	if port < 1 || port > 65535 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": field + " 端口非法（须为 1-65535）: " + strconv.Itoa(port),
		})
		return false
	}
	return true
}

// validateHost 校验目标地址为合法 IP 或域名，非法时写入 400 响应并返回 false。
func validateHost(c *gin.Context, field, host string) bool {
	host = strings.TrimSpace(host)
	if host == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": field + " 不能为空"})
		return false
	}
	if net.ParseIP(host) != nil {
		return true
	}
	// 域名：允许字母数字、点、连字符，不接受空白与协议前缀
	if len(host) > 255 || strings.ContainsAny(host, " \t\r\n/") {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": field + " 格式非法: " + host})
		return false
	}
	return true
}
