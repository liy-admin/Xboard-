import React, { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { getSettings, saveSettings, sendTestMail } from "../../api/config.js";
import { useAuthState } from "../../store/auth.js";
import { Input } from "../../components/Input.jsx";
import { Button } from "../../components/Button.jsx";
import { Switch } from "../../components/Switch.jsx";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/Select.jsx";
import { Dialog } from "../../components/Dialog.jsx";
import debounce from "lodash/debounce";
import isEqual from "lodash/isEqual";

const DEFAULTS = {
    email_host: "", email_port: 465, email_username: "", email_password: "",
    email_encryption: "", email_from_address: "", remind_mail_enable: false,
};

export function EmailSettings() {
    const { t } = useTranslation("settings");
    const auth = useAuthState();
    const [values, setValues] = useState(DEFAULTS);
    const [saving, setSaving] = useState(false);
    const [ready, setReady] = useState(false);
    const lastSavedRef = useRef(null);
    const [testResult, setTestResult] = useState(null);
    const [testOpen, setTestOpen] = useState(false);
    const [testSending, setTestSending] = useState(false);

    useEffect(() => {
        getSettings("email").then((res) => {
            const email = res?.data?.email || res?.data || {};
            const enc = email?.email_encryption;
            setValues({ ...DEFAULTS, ...email, email_encryption: enc || "" });
            lastSavedRef.current = { ...DEFAULTS, ...email, email_encryption: enc || "" };
            setReady(true);
        }).catch(() => setReady(true));
    }, [auth.token]);

    const doSave = useCallback(
        debounce(async (data) => {
            if (isEqual(data, lastSavedRef.current)) return;
            setSaving(true);
            try {
                await saveSettings(data);
                lastSavedRef.current = { ...data };
            } finally { setSaving(false); }
        }, 1000),
        [auth.token],
    );

    const setField = useCallback((name, value) => {
        setValues((prev) => {
            const next = { ...prev, [name]: value };
            doSave(next);
            return next;
        });
    }, [doSave]);

    useEffect(() => () => doSave.cancel(), [doSave]);

    const handleSendTest = async () => {
        setTestSending(true);
        setTestResult(null);
        try {
            const res = await sendTestMail();
            setTestResult(res?.data || res);
            setTestOpen(true);
        } catch (e) {
            setTestResult({ error: e?.message || "Send failed" });
            setTestOpen(true);
        } finally {
            setTestSending(false);
        }
    };

    const encryptionValue = values.email_encryption || "none";

    if (!ready) return null;

    return (
        <>
            <div>
                <h3 className="text-lg font-medium">{t("email.title")}</h3>
                <p className="text-sm text-muted-foreground">{t("email.description")}</p>
                <hr className="my-6" />
                <div className="space-y-4">
                    <FormField label={t("email.email_host.title")} placeholder={t("common.placeholder")} description={t("email.email_host.description")} value={values.email_host ?? ""} onChange={(v) => setField("email_host", v)} />
                    <FormField type="number" label={t("email.email_port.title")} placeholder={t("common.placeholder")} description={t("email.email_port.description")} value={String(values.email_port ?? "")} onChange={(v) => setField("email_port", v ? Number(v) : null)} />
                    <div className="space-y-1.5">
                        <div className="text-base font-medium">{t("email.email_encryption.title")}</div>
                        <Select value={encryptionValue} onValueChange={(v) => setField("email_encryption", v === "none" ? "" : v)}>
                            <SelectTrigger><SelectValue placeholder="Chọn kiểu mã hóa" /></SelectTrigger>
                            <SelectContent>
                                <SelectItem value="none">{t("email.email_encryption.none")}</SelectItem>
                                <SelectItem value="ssl">{t("email.email_encryption.ssl")}</SelectItem>
                                <SelectItem value="tls">{t("email.email_encryption.tls")}</SelectItem>
                            </SelectContent>
                        </Select>
                        <p className="text-sm text-muted-foreground">{t("email.email_encryption.description")}</p>
                    </div>
                    <FormField label={t("email.email_username.title")} placeholder={t("common.placeholder")} description={t("email.email_username.description")} value={values.email_username ?? ""} onChange={(v) => setField("email_username", v)} />
                    <FormField type="password" label={t("email.email_password.title")} placeholder={t("common.placeholder")} description={t("email.email_password.description")} value={values.email_password ?? ""} onChange={(v) => setField("email_password", v)} />
                    <FormField label={t("email.email_from.title")} placeholder={t("common.placeholder")} description={t("email.email_from.description")} value={values.email_from_address ?? ""} onChange={(v) => setField("email_from_address", v)} />
                    <ToggleField label={t("email.remind_mail.title")} description={t("email.remind_mail.description")} checked={values.remind_mail_enable ?? false} onChange={(v) => setField("remind_mail_enable", v)} />
                    <div className="flex items-center justify-between">
                        <Button onClick={handleSendTest} loading={testSending} disabled={testSending}>
                            {testSending ? t("email.test.sending") : t("email.test.title")}
                        </Button>
                    </div>
                </div>
                {saving && <div className="mt-2 text-sm text-muted-foreground">Đang lưu...</div>}
            </div>
            {testOpen && testResult && (
                <Dialog open={testOpen} onOpenChange={setTestOpen} className="sm:max-w-md">
                    <div className="p-6">
                        <h3 className="flex items-center gap-2 text-lg font-semibold">
                            {testResult.error ? (
                                <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="text-destructive"><circle cx="12" cy="12" r="10" /><line x1="12" x2="12" y1="8" y2="12" /><line x1="12" x2="12.01" y1="16" y2="16" /></svg>
                            ) : (
                                <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="text-emerald-500"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" /><path d="m9 11 3 3L22 4" /></svg>
                            )}
                            {testResult.error ? "Gửi email thất bại" : "Gửi email thành công"}
                        </h3>
                        <p className="mt-1 text-sm text-muted-foreground">
                            {testResult.error ? "Đã xảy ra lỗi khi gửi email thử nghiệm" : "Email thử nghiệm đã được gửi thành công, vui lòng kiểm tra hộp thư"}
                        </p>
                        <div className="mt-4 grid gap-2">
                            <div className="grid grid-cols-[100px_1fr] gap-2 text-sm">
                                <div className="text-muted-foreground">Địa chỉ nhận</div>
                                <div>{testResult.email || "-"}</div>
                                <div className="text-muted-foreground">Chủ đề</div>
                                <div>{testResult.subject || "-"}</div>
                                <div className="text-muted-foreground">Tên mẫu</div>
                                <div>{testResult.template_name || "-"}</div>
                            </div>
                            {testResult.error && (
                                <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">{testResult.error}</div>
                            )}
                        </div>
                    </div>
                </Dialog>
            )}
        </>
    );
}

function FormField({ label, placeholder, description, value, onChange, type = "text", as: As = Input }) {
    return (
        <div className="space-y-1.5">
            {label && <div className="text-base font-medium">{label}</div>}
            <As type={type} placeholder={placeholder} value={value} autoComplete="off" onChange={(e) => onChange(e.target.value)} />
            {description && <p className="text-sm text-muted-foreground">{description}</p>}
        </div>
    );
}

function ToggleField({ label, description, checked, onChange }) {
    return (
        <div className="flex flex-row items-center justify-between rounded-lg border p-4">
            <div className="space-y-0.5">
                <div className="text-base font-medium">{label}</div>
                {description && <p className="text-sm text-muted-foreground">{description}</p>}
            </div>
            <Switch checked={checked} onCheckedChange={onChange} />
        </div>
    );
}
