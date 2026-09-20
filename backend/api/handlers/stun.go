package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/netpanel/netpanel/model"
	"github.com/netpanel/netpanel/pkg/logger"
	"github.com/netpanel/netpanel/service/stun"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type StunHandler struct {
	db  *gorm.DB
	log *logrus.Logger
	mgr *stun.Manager
}

func NewStunHandler(db *gorm.DB, log *logrus.Logger, mgr *stun.Manager) *StunHandler {
	return &StunHandler{db: db, log: log, mgr: mgr}
}

func (h *StunHandler) List(c *gin.Context) {
	var rules []model.StunRule
	h.db.Order("id desc").Find(&rules)
	for i := range rules {
		rules[i].Status = h.mgr.GetStatus(rules[i].ID)
		if info := h.mgr.GetCurrentInfo(rules[i].ID); info != nil {
			rules[i].CurrentIP = info.IP
			rules[i].CurrentPort = info.Port
			rules[i].NATType = string(info.NATType)
		}
		if s := h.mgr.GetStunStatus(rules[i].ID); s != "" {
			rules[i].StunStatus = s
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": rules})
}

func (h *StunHandler) Create(c *gin.Context) {
	var rule model.StunRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if strings.TrimSpace(rule.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "规则名称不能为空"})
		return
	}
	switch rule.ForwardMode {
	case "proxy":
		if !validatePort(c, "本地监听端口", rule.ListenPort) {
			return
		}
		if !validateHost(c, "转发目标地址", rule.TargetAddress) {
			return
		}
		if !validatePort(c, "转发目标端口", rule.TargetPort) {
			return
		}
	case "direct":
		if !validatePort(c, "转发目标端口", rule.TargetPort) {
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "转发模式非法（须为 proxy/direct）: " + rule.ForwardMode})
		return
	}
	rule.Status = "stopped"
	h.db.Create(&rule)
	if rule.Enable {
		h.mgr.Start(rule.ID)
	}
	logger.WriteLog("info", "stun", fmt.Sprintf("创建STUN穿透规则 [%d]", rule.ID))
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": rule, "message": "创建成功"})
}

func (h *StunHandler) Update(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req model.StunRule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "规则名称不能为空"})
		return
	}
	switch req.ForwardMode {
	case "proxy":
		if !validatePort(c, "本地监听端口", req.ListenPort) || !validateHost(c, "转发目标地址", req.TargetAddress) || !validatePort(c, "转发目标端口", req.TargetPort) {
			return
		}
	case "direct":
		if !validatePort(c, "转发目标端口", req.TargetPort) {
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "转发模式非法（须为 proxy/direct）: " + req.ForwardMode})
		return
	}
	h.mgr.Stop(id)
	req.ID = id
	h.db.Save(&req)
	if req.Enable {
		h.mgr.Start(id)
	}
	logger.WriteLog("info", "stun", fmt.Sprintf("修改STUN穿透规则 [%d]", id))
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": req, "message": "更新成功"})
}

func (h *StunHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	h.mgr.Stop(uint(id))
	h.db.Delete(&model.StunRule{}, id)
	logger.WriteLog("info", "stun", fmt.Sprintf("删除STUN穿透规则 [%d]", id))
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "删除成功"})
}

func (h *StunHandler) Start(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.mgr.Start(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	h.db.Model(&model.StunRule{}).Where("id = ?", id).Update("enable", true)
	logger.WriteLog("info", "stun", fmt.Sprintf("启动STUN穿透规则 [%d]", id))
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "已启动"})
}

func (h *StunHandler) Stop(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	h.mgr.Stop(uint(id))
	h.db.Model(&model.StunRule{}).Where("id = ?", id).Update("enable", false)
	logger.WriteLog("info", "stun", fmt.Sprintf("停止STUN穿透规则 [%d]", id))
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "已停止"})
}

func (h *StunHandler) GetStatus(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	status := h.mgr.GetStatus(uint(id))
	info := h.mgr.GetCurrentInfo(uint(id))
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{
		"status": status,
		"info":   info,
	}})
}
