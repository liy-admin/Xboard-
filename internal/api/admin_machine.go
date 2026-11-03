package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"xboard/internal/auth"
	"xboard/internal/model"
	"xboard/internal/resp"
)

// 管理端宿主机（machine）：列表、保存、重置令牌、安装命令、删除、节点与负载历史。
// 对应 V2/Admin/Server/MachineController。

// machineInstallerURL 是机器模式一键安装脚本（xnode），可用设置 machine_installer_url 覆盖。
const machineInstallerURL = "https://raw.githubusercontent.com/liy-admin/xnode/dev/install.sh"

// shellQuote 对应 escapeshellarg：单引号包裹并转义内部单引号。
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// machineInstallCommand 对应 buildInstallCommand。
func (a *App) machineInstallCommand(r *http.Request, m model.ServerMachine) string {
	panel := strings.TrimRight(a.Set.Get("app_url", ""), "/")
	if panel == "" {
		panel = strings.TrimRight(schemeHost(r), "/")
	}
	installer := strings.TrimSpace(a.Set.Get("machine_installer_url", ""))
	if installer == "" {
		installer = machineInstallerURL
	}
	return fmt.Sprintf("curl -fsSL %s | sudo bash -s -- --mode machine --panel %s --token %s --machine-id %d",
		installer, shellQuote(panel), shellQuote(m.Token), m.ID)
}

func schemeHost(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func (a *App) machineView(m model.ServerMachine, servers int64) gin.H {
	return gin.H{
		"id": m.ID, "name": m.Name, "notes": m.Notes, "is_active": m.IsActive, "last_seen_at": m.LastSeenAt,
		"load_status": m.LoadStatus, "servers_count": servers, "created_at": m.CreatedAt, "updated_at": m.UpdatedAt,
		"last_ip": a.machineLastIP(m.ID),
	}
}

// machineQueryID 读取查询参数 machine_id，兼容前端传的 id。
func machineQueryID(c *gin.Context) int64 {
	raw := c.Query("machine_id")
	if raw == "" {
		raw = c.Query("id")
	}
	id, _ := strconv.ParseInt(raw, 10, 64)
	return id
}

func (a *App) machineFetch(c *gin.Context) {
	var rows []model.ServerMachine
	a.S.DB.Order("id ASC").Find(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, m := range rows {
		var n int64
		a.S.DB.Model(&model.Server{}).Where("machine_id = ?", m.ID).Count(&n)
		out = append(out, a.machineView(m, n))
	}
	resp.OK(c, out)
}

func (a *App) machineSave(c *gin.Context) {
	m := bodyMap(c)
	name, _ := mStr(m, "name")
	if name == "" {
		resp.Fail(c, 422, "name 不能为空")
		return
	}
	upd := map[string]any{"name": name}
	if notes, ok := m["notes"]; ok {
		if s, ok := notes.(string); ok {
			upd["notes"] = s
		} else {
			upd["notes"] = nil
		}
	}
	active, hasActive := mBool(m, "is_active")
	if id, ok := mInt(m, "id"); ok && id > 0 {
		var mc model.ServerMachine
		if err := a.S.DB.First(&mc, id).Error; err != nil {
			resp.Fail(c, 422, "宿主机不存在")
			return
		}
		if hasActive {
			upd["is_active"] = active
		}
		if err := a.S.DB.Model(&mc).Updates(upd).Error; err != nil {
			resp.Fail(c, 500, "保存失败")
			return
		}
		resp.OK(c, true)
		return
	}
	if !hasActive {
		active = true
	}
	mc := model.ServerMachine{Name: name, IsActive: active, Token: auth.RandHex(32)}
	if s, ok := upd["notes"].(string); ok {
		mc.Notes = &s
	}
	if err := a.S.DB.Create(&mc).Error; err != nil {
		resp.Fail(c, 500, "创建失败")
		return
	}
	resp.OK(c, gin.H{"id": mc.ID, "token": mc.Token, "install_command": a.machineInstallCommand(c.Request, mc)})
}

// machineByID 读取宿主机，不存在时写入失败响应并返回 false。
func (a *App) machineByID(c *gin.Context, id int64) (model.ServerMachine, bool) {
	var m model.ServerMachine
	if err := a.S.DB.First(&m, id).Error; err != nil {
		resp.Fail(c, 422, "宿主机不存在")
		return m, false
	}
	return m, true
}

func (a *App) machineResetToken(c *gin.Context) {
	id, _ := mInt(bodyMap(c), "id")
	m, ok := a.machineByID(c, id)
	if !ok {
		return
	}
	token := auth.RandHex(32)
	a.S.DB.Model(&m).Update("token", token)
	resp.OK(c, gin.H{"token": token})
}

func (a *App) machineGetToken(c *gin.Context) {
	id, _ := mInt(bodyMap(c), "id")
	m, ok := a.machineByID(c, id)
	if !ok {
		return
	}
	resp.OK(c, gin.H{"token": m.Token})
}

func (a *App) machineInstall(c *gin.Context) {
	id, _ := mInt(bodyMap(c), "id")
	m, ok := a.machineByID(c, id)
	if !ok {
		return
	}
	resp.OK(c, gin.H{"command": a.machineInstallCommand(c.Request, m)})
}

// machineDrop 解绑节点（machine_id 置空）后删除宿主机。
func (a *App) machineDrop(c *gin.Context) {
	id, _ := mInt(bodyMap(c), "id")
	m, ok := a.machineByID(c, id)
	if !ok {
		return
	}
	a.S.DB.Model(&model.Server{}).Where("machine_id = ?", m.ID).Update("machine_id", nil)
	a.S.DB.Delete(&m)
	resp.OK(c, true)
}

func (a *App) machineNodesAdmin(c *gin.Context) {
	id := machineQueryID(c)
	if _, ok := a.machineByID(c, id); !ok {
		return
	}
	var rows []model.Server
	a.S.DB.Select("id", "name", "type", "host", "port", "server_port", "protocol_settings", "group_ids", "show", "enabled", "sort", "machine_id").
		Where("machine_id = ?", id).Order("sort ASC").Find(&rows)
	resp.OK(c, rows)
}

// machineHistory 返回负载历史，按时间正序（与 PHP reverse 一致）。
func (a *App) machineHistory(c *gin.Context) {
	id := machineQueryID(c)
	if _, ok := a.machineByID(c, id); !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "60"))
	if limit < 10 || limit > 1440 {
		limit = 60
	}
	q := a.S.DB.Model(&model.ServerMachineLoadHistory{}).Where("machine_id = ?", id)
	if rh, err := strconv.Atoi(c.Query("range_hours")); err == nil && rh >= 1 && rh <= 24 {
		q = q.Where("recorded_at >= ?", now()-int64(rh)*3600)
	}
	var rows []model.ServerMachineLoadHistory
	q.Order("recorded_at DESC").Limit(limit).Find(&rows)
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	resp.OK(c, rows)
}
