import React, { useState, useRef } from "react";
import { Input } from "./Input.jsx";
import { Textarea } from "./Textarea.jsx";
import { Switch } from "./Switch.jsx";
import { Button } from "./Button.jsx";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "./Select.jsx";
import { http, getSecurePath } from "../api/http.js";

/**
 * 把插件路由（如 /api/v1/balala-sub/doctor）拼成【绝对 URL】。
 * 必须绝对，否则 axios 会给它自动加上 baseURL(/api/v2) → /api/v2/api/v1/... 404。
 */
function absUrl(actionUrl) {
    const base = globalThis.window?.settings?.base_url || "/";
    let origin;
    if (/^https?:\/\//i.test(base)) {
        origin = base.replace(/\/+$/, "");
    } else {
        const loc = globalThis.window?.location?.origin || "";
        const sub = base === "/" ? "" : base.replace(/\/+$/, "");
        origin = loc + sub;
    }
    return origin + "/" + String(actionUrl || "").replace(/^\/+/, "");
}

export function configSchemaToValues(schema) {
    return Object.fromEntries(
        Object.entries(schema || {}).map(([key, field]) => {
            let value = field?.value;
            if ((field?.type === "yaml" || field?.type === "json") && value != null && typeof value !== "string") {
                try {
                    value = JSON.stringify(value, null, 2);
                } catch {
                    value = String(value);
                }
            }
            return [key, value ?? ""];
        }),
    );
}

export function DynamicConfigFields({ schema, values, onChange, className = "grid grid-cols-1 gap-4" }) {
    const setField = (key, value) => onChange?.({ ...values, [key]: value });

    return (
        <div className={className}>
            {Object.entries(schema || {}).map(([key, field]) => {
                const label = field.label || field.description;
                const description = field.label ? field.description : undefined;
                const value = values?.[key] ?? "";

                switch (field.type) {
                    case "string":
                        return (
                            <FieldWrap key={key} label={label} description={description} required={field.required}>
                                <Input value={value} placeholder={field.placeholder} onChange={(e) => setField(key, e.target.value)} required={field.required} />
                            </FieldWrap>
                        );
                    case "number":
                    case "percentage":
                        return (
                            <FieldWrap key={key} label={label} description={description} required={field.required}>
                                <div className="relative">
                                    <Input
                                        type="number"
                                        value={value}
                                        placeholder={field.placeholder}
                                        onChange={(e) => {
                                            let n = Number(e.target.value);
                                            if (field.type === "percentage") n = Math.min(100, Math.max(0, n));
                                            setField(key, n);
                                        }}
                                        required={field.required}
                                    />
                                    {field.type === "percentage" && (
                                        <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground">%</span>
                                    )}
                                </div>
                            </FieldWrap>
                        );
                    case "select":
                        return (
                            <FieldWrap key={key} label={label} description={description} required={field.required}>
                                <Select value={String(value ?? "")} onValueChange={(v) => setField(key, v)}>
                                    <SelectTrigger><SelectValue placeholder={field.placeholder} /></SelectTrigger>
                                    <SelectContent>
                                        {(field.options || []).map((opt) => (
                                            <SelectItem key={opt.value} value={String(opt.value)}>{opt.label}</SelectItem>
                                        ))}
                                    </SelectContent>
                                </Select>
                            </FieldWrap>
                        );
                    case "boolean":
                        return (
                            <div key={key} className="flex flex-row items-center justify-between rounded-lg border p-4">
                                <div className="space-y-0.5">
                                    <div className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                                        {label}
                                        {field.required && <span className="ml-1 text-destructive">*</span>}
                                    </div>
                                    {description && <p className="text-xs text-muted-foreground">{description}</p>}
                                </div>
                                <Switch checked={Boolean(value)} onCheckedChange={(v) => setField(key, v)} />
                            </div>
                        );
                    case "text":
                        return (
                            <FieldWrap key={key} label={label} description={description} required={field.required}>
                                <Textarea className="min-h-[80px] font-mono text-xs" value={value} placeholder={field.placeholder} onChange={(e) => setField(key, e.target.value)} required={field.required} />
                            </FieldWrap>
                        );
                    case "yaml":
                    case "json":
                        return (
                            <FieldWrap key={key} label={label} description={description} required={field.required}>
                                <Textarea className="min-h-[120px] font-mono text-xs" value={value} onChange={(e) => setField(key, e.target.value)} />
                            </FieldWrap>
                        );
                    case "file":
                        return (
                            <FieldWrap key={key} label={label} description={description} required={field.required}>
                                <FileField field={field} value={value} onChange={(v) => setField(key, v)} />
                            </FieldWrap>
                        );
                    case "action":
                        return (
                            <FieldWrap key={key} label={label} description={description} required={field.required}>
                                <ActionField field={field} />
                            </FieldWrap>
                        );
                    default:
                        return null;
                }
            })}
        </div>
    );
}

/**
 * 文件上传字段：显示/可编辑一个服务器路径，并提供上传按钮。
 * 上传到 field.upload_url（相对站点根的路径），后端处理后返回 { data: { path } }，
 * 该 path 作为字段值保存。用于「上传 Firefox 内核 zip」等场景。
 */
function FileField({ field, value, onChange }) {
    const [uploading, setUploading] = useState(false);
    const [error, setError] = useState("");
    const [note, setNote] = useState("");
    const ref = useRef(null);

    const upload = async (file) => {
        if (!file) return;
        setUploading(true);
        setError("");
        setNote("");
        try {
            const fd = new FormData();
            fd.append("file", file);
            const url = absUrl(field.upload_url);
            const res = await http.post(url, fd, {
                headers: { "Content-Type": undefined },
                timeout: 600000,
            });
            const path = res?.data?.path || res?.data?.chrome_path || "";
            if (path) {
                onChange(path);
                setNote(res?.message || "上传成功");
            } else {
                setError(res?.message || "上传失败：未返回路径");
            }
        } catch (e) {
            setError(e?.message || String(e));
        } finally {
            setUploading(false);
            if (ref.current) ref.current.value = "";
        }
    };

    return (
        <div className="space-y-1.5">
            <div className="flex items-center gap-2">
                <Input
                    value={value || ""}
                    placeholder={field.placeholder}
                    onChange={(e) => onChange(e.target.value)}
                    className="flex-1"
                />
                <input
                    ref={ref}
                    type="file"
                    accept={field.accept || ""}
                    className="hidden"
                    onChange={(e) => e.target.files?.[0] && upload(e.target.files[0])}
                />
                <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    loading={uploading}
                    onClick={() => ref.current?.click()}
                >
                    {uploading ? (field.uploading_text || "上传中...") : (field.upload_text || "上传")}
                </Button>
            </div>
            {note && <p className="text-xs text-emerald-600">{note}</p>}
            {error && <p className="text-xs text-destructive">{error}</p>}
        </div>
    );
}

/**
 * 动作按钮字段：点击调用 field.action_url（GET），展示返回结果。
 * 若返回数据含 checks 数组（如环境自检），逐项渲染 ✓/✗；否则显示 message。
 */
function ActionField({ field }) {
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState("");
    const [result, setResult] = useState(null);

    const run = async () => {
        setLoading(true);
        setError("");
        setResult(null);
        try {
            // 以 http(https) 或 / 开头 → 绝对 URL（插件自有路由）；否则视为后台 api/v2 相对路径，
            // 走和其它后台请求完全相同的通道：baseURL(/api/v2) + secure_path + 路径。最稳。
            const au = String(field.action_url || "");
            const isAbsolute = /^https?:\/\//i.test(au) || au.startsWith("/");
            const url = isAbsolute ? absUrl(au) : getSecurePath() + "/" + au.replace(/^\/+/, "");
            const res = await http.get(url, { timeout: 180000 });
            // success() 信封是 {status,message,data:{ok,checks}}；也兼容直接返回 {ok,checks}
            const inner = res && typeof res === "object" ? res.data : null;
            const payload =
                inner && (Array.isArray(inner.checks) || typeof inner.ok !== "undefined" || typeof inner.message !== "undefined")
                    ? inner
                    : res;
            setResult(payload ?? { message: "完成" });
        } catch (e) {
            setError(e?.message || (typeof e === "string" ? e : JSON.stringify(e)));
        } finally {
            setLoading(false);
        }
    };

    const checks = Array.isArray(result?.checks) ? result.checks : null;

    return (
        <div className="space-y-2">
            <Button type="button" variant="outline" size="sm" loading={loading} onClick={run}>
                {loading ? (field.running_text || "检查中...") : (field.action_text || "运行")}
            </Button>
            {error && <p className="text-xs text-destructive">{error}</p>}
            {checks && (
                <div className="rounded-lg border p-3 text-xs">
                    <div className={`mb-2 font-medium ${result.ok ? "text-emerald-600" : "text-destructive"}`}>
                        {result.ok ? "✅ 环境正常" : "❌ 存在问题"}
                    </div>
                    <ul className="space-y-1">
                        {checks.map((c, i) => (
                            <li key={i} className="flex gap-2">
                                <span className={c.ok ? "text-emerald-600" : "text-destructive"}>{c.ok ? "✓" : "✗"}</span>
                                <span className="font-medium">{c.name}</span>
                                <span className="text-muted-foreground">{c.detail}</span>
                            </li>
                        ))}
                    </ul>
                </div>
            )}
            {result && !checks && (
                result.message
                    ? <p className="text-xs text-muted-foreground">{result.message}</p>
                    : <pre className="overflow-auto rounded-lg border p-3 text-[10px] text-muted-foreground">{JSON.stringify(result, null, 2)}</pre>
            )}
        </div>
    );
}

function FieldWrap({ label, description, required, children }) {
    return (
        <div className="space-y-1.5">
            <div className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                {label}
                {required && <span className="ml-1 text-destructive">*</span>}
            </div>
            {children}
            {description && <p className="text-xs text-muted-foreground">{description}</p>}
        </div>
    );
}
