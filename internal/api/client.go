package api

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"xboard/internal/model"
	"xboard/internal/resp"
	"xboard/internal/shop"
	"xboard/internal/sub"
)

func (a *App) clientAuth(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		token = c.Param("token")
	}
	if token == "" {
		resp.Fail(c, 403, "token is null")
		return
	}
	var u model.User
	if err := a.S.DB.Where("token = ?", token).First(&u).Error; err != nil {
		resp.Fail(c, 403, "token is error")
		return
	}
	c.Set(userKey, &u)
	c.Next()
}

// 明文订阅已按原实现下线：只保留 403，避免通过 flag 读取原始节点凭据。
func (a *App) subscribePlain(c *gin.Context) {
	c.Data(403, "text/plain", nil)
}

func userAvailable(u *model.User) bool {
	return shop.IsAvailable(u) // 流量为 0 视为无限流量
}

func (a *App) template(name string) string {
	var t model.SubscribeTemplate
	if a.S.DB.Where("name = ?", name).First(&t).Error == nil && t.Content != nil {
		return *t.Content
	}
	return ""
}

// doSubscribe 对应 ClientController::doSubscribe，返回客户端内容与响应头。
func (a *App) doSubscribe(c *gin.Context, u *model.User, flag string) (body []byte, ctype string, hdr map[string]string, err error) {
	servers, err := sub.AvailableServers(c, a.S.DB, a.S.Redis, u)
	if err != nil {
		return nil, "", nil, err
	}
	// 与后台「是否下发」共用同一套筛选（见 computeDeliveredIDs），两边结论保持一致。
	servers = sub.SelectForDelivery(servers, a.Set.Bool("server_dedup_by_country", false), a.deliveryStale(),
		a.Set.Bool("server_country_naming", true), sub.CountryDisplayName)
	ci := sub.ParseClient(flag)
	filtered := sub.FilterServers(servers, sub.ParseTypes(c.Query("types")), sub.ParseFilter(c.Query("filter")))

	ctx := &sub.Ctx{
		User: u, ClientName: ci.Name, ClientVersion: ci.Version, UserAgent: ci.Flag,
		AppName:  a.Set.Get("app_name", "Lets VPN"),
		Template: a.template,
	}
	filtered = sub.AddInfoServers(filtered, ctx, len(servers)-len(filtered), a.Set.Int("show_info_to_server_enable", 0) != 0)
	if a.Set.Int("show_protocol_to_server_enable", 0) != 0 {
		filtered = sub.AddPrefix(filtered)
	}
	ctx.Servers = filtered

	ctx.Host = c.Request.Host
	ctx.SubscribeURL = a.subscribeURL
	ctx.AppURL = a.Set.Get("app_url", "")
	name, ported := sub.MatchProtocol(ci.Flag)
	if ported {
		switch name {
		case "clash", "clashmeta", "stash":
			kind := map[string]sub.ClashKind{"clash": sub.ClashKindClash, "clashmeta": sub.ClashKindMeta, "stash": sub.ClashKindStash}[name]
			out, err := sub.BuildClash(ctx, kind)
			if err != nil {
				return nil, "", nil, err
			}
			return out.Body, "text/yaml", out.Headers, nil
		}
	}
	if ported {
		switch name {
		case "surge":
			body, ct, h := sub.BuildSurge(ctx)
			return []byte(body), ct, h, nil
		case "surfboard":
			body, ct, h := sub.BuildSurfboard(ctx)
			return []byte(body), ct, h, nil
		case "loon":
			body, ct, h := sub.BuildLoon(ctx)
			return []byte(body), ct, h, nil
		case "quantumultx":
			body, ct, h := sub.BuildQuantumultX(ctx)
			return []byte(body), ct, h, nil
		case "shadowrocket":
			body, ct, h := sub.BuildShadowrocket(ctx)
			return []byte(body), ct, h, nil
		case "shadowsocks":
			body, ct := sub.BuildSIP008(ctx)
			return []byte(body), ct, nil, nil
		}
	}
	if name == "singbox" && ported {
		cfg, h, err := sub.BuildSingBox(ctx)
		if err != nil {
			return nil, "", nil, err
		}
		b, _ := json.Marshal(cfg)
		return b, "application/json", h, nil
	}
	s, ct, h := sub.BuildGeneral(ctx)
	return []byte(s), ct, h, nil
}

// subscribeEncrypted: 强制 sing-box 输出，AES-256-GCM，key = SHA256(enc_key)。
func (a *App) subscribeEncrypted(c *gin.Context) {
	u := cur(c)
	if !userAvailable(u) {
		c.Data(403, "text/plain", nil)
		return
	}
	plain, _, _, err := a.doSubscribe(c, u, "sing-box")
	if err != nil {
		c.Data(500, "text/plain", nil)
		return
	}
	enc := a.encKey(u)
	key := sha256.Sum256([]byte(enc))
	blk, err := aes.NewCipher(key[:])
	if err != nil {
		c.Data(500, "text/plain", nil)
		return
	}
	gcm, _ := cipher.NewGCM(blk)
	nonce := make([]byte, gcm.NonceSize())
	_, _ = rand.Read(nonce)
	sealed := gcm.Seal(nil, nonce, plain, nil)
	n := len(sealed) - gcm.Overhead()
	e := base64.StdEncoding
	c.JSON(http.StatusOK, gin.H{
		"v": 1, "alg": "aes-256-gcm",
		"n": e.EncodeToString(nonce), "c": e.EncodeToString(sealed[:n]), "t": e.EncodeToString(sealed[n:]),
	})
}

func (a *App) registerClientRoutes(r *gin.Engine) {
	path := strings.Trim(a.Set.Get("subscribe_path", "s"), "/")
	r.GET("/"+path+"/:token", a.clientAuth, a.subscribePlain)
	r.GET("/"+path+"-enc/:token", a.clientAuth, a.subscribeEncrypted)
	r.GET("/api/v1/client/subscribe", a.clientAuth, a.subscribePlain)
}
