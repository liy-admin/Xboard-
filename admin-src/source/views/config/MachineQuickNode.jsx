import React, { useEffect, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { fetchGroups, quickCreateMachineNode } from "../../api/server.js";
import { getErrorMessage } from "../../utils/errors.js";
import { Button } from "../../components/Button.jsx";
import { Input } from "../../components/Input.jsx";
import { Switch } from "../../components/Switch.jsx";

const DEFAULT_PORT = "443";
const DEFAULT_SNI = "www.apple.com";

// 服务器详情里的「一键创建节点」：选权限组即可创建绑定到该服务器的 VLESS + Reality 节点，
// Reality 密钥由面板生成。地址缺省为服务器最近连接面板的 IP（last_ip）。
export function MachineQuickNode({ machine, onCreated, onCancel }) {
    const { t } = useTranslation("machine");
    const [groups, setGroups] = useState(null);
    const [groupIds, setGroupIds] = useState([]);
    const [host, setHost] = useState(machine.last_ip || "");
    const [port, setPort] = useState(DEFAULT_PORT);
    const [name, setName] = useState("");
    const [sni, setSni] = useState(DEFAULT_SNI);
    const [show, setShow] = useState(true);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState("");
    const [created, setCreated] = useState(null);

    useEffect(() => {
        fetchGroups()
            .then((res) => {
                const list = res?.data || [];
                setGroups(list);
                if (list.length === 1) setGroupIds([list[0].id]);
            })
            .catch((err) => {
                setGroups([]);
                setError(getErrorMessage(err));
            });
    }, []);

    const toggleGroup = (id, checked) => {
        setGroupIds((prev) => (checked ? [...prev, id] : prev.filter((g) => g !== id)));
    };

    const submit = async () => {
        setError("");
        if (groupIds.length === 0) {
            setError(t("quickNode.needGroup"));
            return;
        }
        setSubmitting(true);
        try {
            const res = await quickCreateMachineNode({
                machine_id: machine.id,
                group_ids: groupIds,
                host: host.trim(),
                port: Number(port) || Number(DEFAULT_PORT),
                name: name.trim() || undefined,
                server_name: sni.trim() || undefined,
                show,
            });
            setCreated(res?.data || null);
            onCreated?.(res?.data);
        } catch (err) {
            setError(getErrorMessage(err));
        } finally {
            setSubmitting(false);
        }
    };

    if (created) {
        return (
            <div className="mt-3 rounded-lg border border-emerald-500/30 bg-emerald-500/5 p-4 text-sm">
                <p className="font-medium text-emerald-700 dark:text-emerald-400">
                    {t("quickNode.created", { id: created.id, host: created.host, port: created.port })}
                </p>
                <p className="mt-1 text-xs text-muted-foreground">{t("quickNode.createdHint")}</p>
                <div className="mt-3 flex justify-end">
                    <Button variant="outline" size="sm" onClick={onCancel}>{t("quickNode.close")}</Button>
                </div>
            </div>
        );
    }

    return (
        <div className="mt-3 space-y-3 rounded-lg border p-4">
            <div>
                <h5 className="text-sm font-medium">{t("quickNode.title")}</h5>
                <p className="mt-1 text-xs text-muted-foreground">{t("quickNode.description")}</p>
            </div>

            {error && <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">{error}</div>}

            <div className="space-y-1.5">
                <div className="text-sm font-medium">{t("quickNode.groups")}</div>
                {groups === null ? (
                    <p className="text-xs text-muted-foreground">...</p>
                ) : groups.length === 0 ? (
                    <p className="text-xs text-muted-foreground">{t("quickNode.noGroups")}</p>
                ) : (
                    <div className="flex flex-wrap gap-x-4 gap-y-1.5">
                        {groups.map((g) => (
                            <label key={g.id} className="inline-flex cursor-pointer items-center gap-1.5 text-sm">
                                <input type="checkbox" checked={groupIds.includes(g.id)} onChange={(e) => toggleGroup(g.id, e.target.checked)} />
                                <span>{g.name}</span>
                                <span className="font-mono text-xs text-muted-foreground">#{g.id}</span>
                            </label>
                        ))}
                    </div>
                )}
            </div>

            <div className="space-y-1.5">
                <div className="text-sm font-medium">{t("quickNode.host")}</div>
                <Input value={host} placeholder="203.0.113.10 / node.example.com" onChange={(e) => setHost(e.target.value)} />
                <p className="text-xs text-muted-foreground">{machine.last_ip ? t("quickNode.hostHint") : t("quickNode.hostMissing")}</p>
            </div>

            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("quickNode.port")}</div>
                    <Input type="number" min={1} max={65535} value={port} onChange={(e) => setPort(e.target.value)} />
                </div>
                <div className="space-y-1.5">
                    <div className="text-sm font-medium">{t("quickNode.name")}</div>
                    <Input value={name} placeholder={machine.name} onChange={(e) => setName(e.target.value)} />
                </div>
            </div>

            <div className="space-y-1.5">
                <div className="text-sm font-medium">{t("quickNode.sni")}</div>
                <Input value={sni} placeholder={DEFAULT_SNI} onChange={(e) => setSni(e.target.value)} />
                <p className="text-xs text-muted-foreground">{t("quickNode.sniHint")}</p>
            </div>

            <div className="flex items-center justify-between rounded-lg border p-3">
                <div className="text-sm font-medium">{t("quickNode.show")}</div>
                <Switch checked={show} onCheckedChange={setShow} />
            </div>

            <div className="flex justify-end gap-2">
                <Button variant="outline" onClick={onCancel} disabled={submitting}>{t("quickNode.cancel")}</Button>
                <Button onClick={submit} disabled={submitting || groups === null}>
                    {submitting ? t("quickNode.creating") : t("quickNode.submit")}
                </Button>
            </div>
        </div>
    );
}
