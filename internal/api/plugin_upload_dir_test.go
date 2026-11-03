package api

import "testing"

func TestPluginUploadDirIsNeverTheCoreDirectory(t *testing.T) {
	cases := []struct{ env, want string }{
		{"", "plugins"},                   // 默认 plugins-core,plugins：上传必须进 plugins
		{"plugins-core", "plugins"},       // 只配了核心目录：回退到 plugins
		{"plugins-core,custom", "custom"}, // 自定义目录
		{"/opt/x/plugins-core,/opt/x/plugins", "/opt/x/plugins"},
		{"plugins,plugins-core", "plugins"},
	}
	for _, c := range cases {
		t.Setenv("XBOARD_PLUGINS_DIR", c.env)
		if got := pluginUploadDir(); got != c.want {
			t.Errorf("XBOARD_PLUGINS_DIR=%q => %q, want %q", c.env, got, c.want)
		}
	}
}
