package sub

import "time"

func fmtDate(ts int64) string { return time.Unix(ts, 0).Format("2006-01-02") }
