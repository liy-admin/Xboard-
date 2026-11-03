package api

import (
	"strings"

	"github.com/gin-gonic/gin"

	"xboard/internal/jobs"
	"xboard/internal/model"
	"xboard/internal/resp"
	"xboard/internal/singbox"
)

// probeDiagnose 对应 ManageController::probeDiagnose / NodeProbeService::diagnose：
// 检测 sing-box 是否可用（自定义路径 → PATH），并对一个样本节点实跑一次。
// 只定位不下载；找不到时返回 ok=false 并提示使用「在线下载」。
func (a *App) probeDiagnose(c *gin.Context) {
	m := bodyMap(c)
	mode := a.Set.Get("node_probe_mode", "singbox")
	sb := gin.H{"ok": false, "path": nil, "version": nil, "error": nil}
	bin, ver, errMsg := a.jobsEnv().ResolveSingBoxFor()
	if bin != "" {
		sb["ok"], sb["path"], sb["version"] = true, bin, ver
	} else {
		sb["error"] = "无法启动 sing-box：" + errMsg + "。请点击下方「在线下载」，或手动安装 sing-box 并在 node_probe_singbox_path 指定路径"
	}

	var sample any
	var node model.Server
	q := a.S.DB.Where("parent_id IS NULL")
	var err error
	if id, ok := mInt(m, "id"); ok && id > 0 {
		err = q.First(&node, id).Error
	} else if err = q.Where("`show` = 1").First(&node).Error; err != nil {
		err = a.S.DB.Where("parent_id IS NULL").First(&node).Error
	}
	if err == nil {
		env := a.jobsEnv()
		var r jobs.ProbeResult
		if mode == "singbox" && bin == "" {
			msg := "sing-box 不可用，未执行样本探测"
			r = jobs.ProbeResult{Error: &msg, Untestable: true}
		} else {
			// 诊断用快速探测：1 次、请求超时 5 秒。
			r = env.ProbeNode(node, jobs.ProbeOptions{TestCount: 1, RequestTimeoutMs: 5000})
		}
		sample = gin.H{"node_id": node.ID, "name": node.Name, "host": node.Host, "result": r}
	}
	resp.OK(c, gin.H{"mode": mode, "singbox": sb, "sample": sample})
}

// singboxDownload 开始在线下载 sing-box（异步，立即返回当前状态；用 singboxDownloadStatus 轮询进度）。
// body.version 缺省时取后台设置 node_probe_singbox_version。
func (a *App) singboxDownload(c *gin.Context) {
	m := bodyMap(c)
	version, _ := mStr(m, "version")
	version = strings.TrimSpace(strings.TrimPrefix(version, "v"))
	if version == "" {
		version = strings.TrimSpace(a.Set.Get("node_probe_singbox_version", singbox.DefaultVersion))
	}
	st, err := singbox.Default().Start(version)
	if err != nil {
		resp.Fail(c, 422, err.Error())
		return
	}
	resp.OK(c, st)
}

// singboxDownloadStatus 返回下载进度：state 为 idle / downloading / done / error。
func (a *App) singboxDownloadStatus(c *gin.Context) {
	resp.OK(c, singbox.Default().Status())
}
