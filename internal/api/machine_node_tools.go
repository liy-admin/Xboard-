package api

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"xboard/internal/jobs"
	"xboard/internal/model"
	"xboard/internal/resp"
	"xboard/internal/sub"
)

// 服务器详情里的节点工具：实测节点是否可用、导出分享链接。

// machineNodeTest 同步实测一个节点：sing-box 模式下经节点真实访问外网（与定时测速同一套逻辑），
// TCP 模式下只测端口可达。测的是「面板服务器 → 节点」这条路，不代表用户所在网络一定能连上。
func (a *App) machineNodeTest(c *gin.Context) {
	id, _ := mInt(bodyMap(c), "id")
	var node model.Server
	if id <= 0 || a.S.DB.First(&node, id).Error != nil {
		resp.Fail(c, 400, "节点不存在")
		return
	}
	env := a.jobsEnv()
	mode := a.Set.Get("node_probe_mode", "singbox")
	if mode == "singbox" {
		if bin, _, errMsg := env.ResolveSingBoxFor(); bin == "" {
			msg := "面板上没有可用的 sing-box（" + errMsg + "），无法实测。请到「节点管理」的测速设置里在线下载 sing-box"
			resp.OK(c, gin.H{"mode": mode, "result": jobs.ProbeResult{Error: &msg, Untestable: true}})
			return
		}
	}
	r := env.ProbeNode(node, jobs.ProbeOptions{TestCount: 3, RequestTimeoutMs: 8000})
	resp.OK(c, gin.H{"mode": mode, "result": r})
}

type nodeLink struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Enabled   bool   `json:"enabled"`
	Link      string `json:"link"`
	UserEmail string `json:"user_email"`
	Error     string `json:"error,omitempty"`
	Warning   string `json:"warning,omitempty"`
}

// machineNodeLinks 生成服务器上全部节点的分享链接。传 email 时用该用户的凭据，
// 否则每个节点用其权限组里第一个可用用户（与测速相同的规则）。
func (a *App) machineNodeLinks(c *gin.Context) {
	mid := machineQueryID(c)
	if _, ok := a.machineByID(c, mid); !ok {
		return
	}
	var user *model.User
	if email := strings.TrimSpace(c.Query("email")); email != "" {
		var u model.User
		if a.S.DB.Where("email = ?", email).First(&u).Error != nil {
			resp.Fail(c, 400, "用户不存在："+email)
			return
		}
		user = &u
	}

	var nodes []model.Server
	a.S.DB.Where("machine_id = ?", mid).Order("sort ASC, id ASC").Find(&nodes)
	env := a.jobsEnv()
	out := make([]nodeLink, 0, len(nodes))
	for _, n := range nodes {
		item := nodeLink{ID: n.ID, Name: n.Name, Type: n.Type, Enabled: n.Enabled == nil || *n.Enabled}
		u := user
		if u == nil {
			found, ok, why := env.FindProbeUser(n)
			if !ok {
				item.Error = why
				out = append(out, item)
				continue
			}
			u = found
		} else if !userInNodeGroups(u, n) {
			item.Warning = "该用户不在节点的权限组里，节点不会接受这个链接"
		}
		parent := n
		if n.ParentID != nil && *n.ParentID > 0 {
			a.S.DB.First(&parent, *n.ParentID)
		}
		item.UserEmail = u.Email
		item.Link = sub.ShareLink(n, parent, u)
		if item.Link == "" {
			item.Error = "协议 " + n.Type + " 没有通用的分享链接格式"
		}
		out = append(out, item)
	}
	resp.OK(c, out)
}

func userInNodeGroups(u *model.User, n model.Server) bool {
	if u.GroupID == nil {
		return false
	}
	var raw []any
	_ = json.Unmarshal(n.GroupIDs, &raw)
	for _, v := range raw {
		if id, err := strconv.ParseInt(strings.TrimSpace(sub.Str(v)), 10, 64); err == nil && id == *u.GroupID {
			return true
		}
	}
	return false
}
