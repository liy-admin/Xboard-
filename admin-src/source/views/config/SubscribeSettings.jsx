import React, { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { getSettings, saveSettings } from "../../api/config.js";
import { useAuthState } from "../../store/auth.js";
import { Switch } from "../../components/Switch.jsx";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/Select.jsx";
import debounce from "lodash/debounce";
import isEqual from "lodash/isEqual";

const DEFAULTS = {
    plan_change_enable: false, reset_traffic_method: 0, surplus_enable: false,
    new_order_event_id: 0, renew_order_event_id: 0, change_order_event_id: 0,
    show_info_to_server_enable: false, show_protocol_to_server_enable: false,
    default_remind_expire: false, default_remind_traffic: false, subscribe_path: "s",
};

export function SubscribeSettings() {
    const { t } = useTranslation("settings");
    const auth = useAuthState();
    const [values, setValues] = useState(DEFAULTS);
    const [saving, setSaving] = useState(false);
    const [ready, setReady] = useState(false);
    const lastSavedRef = useRef(null);

    useEffect(() => {
        getSettings("subscribe").then((res) => {
            const sub = res?.data?.subscribe || res?.data || {};
            const next = { ...DEFAULTS, ...sub };
            setValues(next);
            lastSavedRef.current = { ...next };
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

    if (!ready) return null;

    return (
        <div>
            <h3 className="text-lg font-medium">{t("subscribe.title")}</h3>
            <p className="text-sm text-muted-foreground">{t("subscribe.description")}</p>
            <hr className="my-6" />
            <div className="space-y-4">
                <div className="space-y-1.5">
                    <div className="text-base font-medium">{t("subscribe.plan_change_enable.title")}</div>
                    <p className="text-sm text-muted-foreground">{t("subscribe.plan_change_enable.description")}</p>
                    <Switch checked={values.plan_change_enable} onCheckedChange={(v) => setField("plan_change_enable", v)} />
                </div>
                <div className="space-y-1.5">
                    <div className="text-base font-medium">{t("subscribe.reset_traffic_method.title")}</div>
                    <Select value={String(values.reset_traffic_method ?? "0")} onValueChange={(v) => setField("reset_traffic_method", Number(v))}>
                        <SelectTrigger><SelectValue placeholder="Chọn phương thức reset" /></SelectTrigger>
                        <SelectContent>
                            <SelectItem value="0">{t("subscribe.reset_traffic_method.options.monthly_first")}</SelectItem>
                            <SelectItem value="1">{t("subscribe.reset_traffic_method.options.monthly_reset")}</SelectItem>
                            <SelectItem value="2">{t("subscribe.reset_traffic_method.options.no_reset")}</SelectItem>
                            <SelectItem value="3">{t("subscribe.reset_traffic_method.options.yearly_first")}</SelectItem>
                            <SelectItem value="4">{t("subscribe.reset_traffic_method.options.yearly_reset")}</SelectItem>
                        </SelectContent>
                    </Select>
                    <p className="text-sm text-muted-foreground">{t("subscribe.reset_traffic_method.description")}</p>
                </div>
                <div className="space-y-1.5">
                    <div className="text-base font-medium">{t("subscribe.surplus_enable.title")}</div>
                    <p className="text-sm text-muted-foreground">{t("subscribe.surplus_enable.description")}</p>
                    <Switch checked={values.surplus_enable} onCheckedChange={(v) => setField("surplus_enable", v)} />
                </div>
                <div className="space-y-1.5">
                    <div className="text-base font-medium">{t("subscribe.new_order_event.title")}</div>
                    <div className="relative w-max">
                        <Select value={String(values.new_order_event_id ?? "0")} onValueChange={(v) => setField("new_order_event_id", Number(v))}>
                            <SelectTrigger><SelectValue placeholder="Chọn" /></SelectTrigger>
                            <SelectContent>
                                <SelectItem value="0">{t("subscribe.new_order_event.options.no_action")}</SelectItem>
                                <SelectItem value="1">{t("subscribe.new_order_event.options.reset_traffic")}</SelectItem>
                            </SelectContent>
                        </Select>
                    </div>
                    <p className="text-sm text-muted-foreground">{t("subscribe.new_order_event.description")}</p>
                </div>
                <div className="space-y-1.5">
                    <div className="text-base font-medium">{t("subscribe.renew_order_event.title")}</div>
                    <div className="relative w-max">
                        <Select value={String(values.renew_order_event_id ?? "0")} onValueChange={(v) => setField("renew_order_event_id", Number(v))}>
                            <SelectTrigger><SelectValue placeholder="Chọn" /></SelectTrigger>
                            <SelectContent>
                                <SelectItem value="0">{t("subscribe.renew_order_event.options.no_action")}</SelectItem>
                                <SelectItem value="1">{t("subscribe.renew_order_event.options.reset_traffic")}</SelectItem>
                            </SelectContent>
                        </Select>
                    </div>
                    <p className="text-sm text-muted-foreground">{t("subscribe.renew_order_event.description")}</p>
                </div>
                <div className="space-y-1.5">
                    <div className="text-base font-medium">{t("subscribe.change_order_event.title")}</div>
                    <div className="relative w-max">
                        <Select value={String(values.change_order_event_id ?? "0")} onValueChange={(v) => setField("change_order_event_id", Number(v))}>
                            <SelectTrigger><SelectValue placeholder="Chọn" /></SelectTrigger>
                            <SelectContent>
                                <SelectItem value="0">{t("subscribe.change_order_event.options.no_action")}</SelectItem>
                                <SelectItem value="1">{t("subscribe.change_order_event.options.reset_traffic")}</SelectItem>
                            </SelectContent>
                        </Select>
                    </div>
                    <p className="text-sm text-muted-foreground">{t("subscribe.change_order_event.description")}</p>
                </div>
                <div className="space-y-1.5">
                    <div className="text-base font-medium">{t("subscribe.show_info_to_server.title")}</div>
                    <p className="text-sm text-muted-foreground">{t("subscribe.show_info_to_server.description")}</p>
                    <Switch checked={values.show_info_to_server_enable} onCheckedChange={(v) => setField("show_info_to_server_enable", v)} />
                </div>
                <div className="space-y-1.5">
                    <div className="text-base font-medium">{t("subscribe.show_protocol_to_server.title")}</div>
                    <p className="text-sm text-muted-foreground">{t("subscribe.show_protocol_to_server.description")}</p>
                    <Switch checked={values.show_protocol_to_server_enable} onCheckedChange={(v) => setField("show_protocol_to_server_enable", v)} />
                </div>
                {saving && <div className="text-sm text-muted-foreground">Đang lưu...</div>}
            </div>
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
