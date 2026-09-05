import React, { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { useAuthState } from "../../store/auth.js";
import {
    getPluginTypes,
    getPluginList,
    uploadPlugin,
    deletePlugin,
    installPlugin,
    uninstallPlugin,
    enablePlugin,
    disablePlugin,
    getPluginConfig,
    updatePluginConfig,
    upgradePlugin,
} from "../../api/plugin.js";
import { Button } from "../../components/Button.jsx";
import { Input } from "../../components/Input.jsx";
import { Badge } from "../../components/Badge.jsx";
import { Card } from "../../components/Card.jsx";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "../../components/Tabs.jsx";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../components/Select.jsx";
import { Dialog } from "../../components/Dialog.jsx";
import { FormDialog } from "../../components/FormDialog.jsx";
import { ConfirmDialog } from "../../components/ConfirmDialog.jsx";
import { DynamicConfigFields, configSchemaToValues } from "../../components/DynamicConfigFields.jsx";
import { classNames } from "../../utils/classNames.js";

export function PluginManagement() {
    const { t } = useTranslation("plugin");
    const auth = useAuthState();
    const [types, setTypes] = useState([]);
    const [plugins, setPlugins] = useState([]);
    const [loading, setLoading] = useState(true);
    const [search, setSearch] = useState("");
    const [typeFilter, setTypeFilter] = useState("all");
    const [statusFilter, setStatusFilter] = useState("all");
    const [actionCode, setActionCode] = useState(null);

    const [uploadOpen, setUploadOpen] = useState(false);
    const [uploading, setUploading] = useState(false);
    const [dragOver, setDragOver] = useState(false);
    const [uploadError, setUploadError] = useState("");
    const fileRef = useRef(null);

    const [readmeOpen, setReadmeOpen] = useState(false);
    const [readmeContent, setReadmeContent] = useState("");
    const [configTarget, setConfigTarget] = useState(null);
    const [deleteTarget, setDeleteTarget] = useState(null);
    const [uninstallTarget, setUninstallTarget] = useState(null);
    const [upgradeTarget, setUpgradeTarget] = useState(null);

    const load = () => {
        setLoading(true);
        return getPluginList()
            .then((res) => setPlugins(res?.data || []))
            .finally(() => setLoading(false));
    };

    useEffect(() => {
        getPluginTypes().then((res) => setTypes(res?.data || []));
        load();
    }, [auth.token]);

    const typeInfo = (value) => types.find((tp) => tp.value === value);

    const filtered = useMemo(() => {
        const keyword = search.trim().toLowerCase();
        let list = plugins;
        if (keyword) {
            list = list.filter((p) =>
                (p.name || "").toLowerCase().includes(keyword)
                || (p.description || "").toLowerCase().includes(keyword)
                || (p.code || "").toLowerCase().includes(keyword),
            );
        }
        if (typeFilter !== "all") list = list.filter((p) => p.type === typeFilter);
        if (statusFilter === "installed") list = list.filter((p) => p.is_installed);
        else if (statusFilter === "available") list = list.filter((p) => !p.is_installed);
        return list;
    }, [plugins, search, typeFilter, statusFilter]);

    const runAction = async (code, fn, successKey, errorKey) => {
        setActionCode(code);
        try {
            await fn();
            await load();
        } catch (e) {
            setUploadError(e?.message || t(errorKey));
        } finally {
            setActionCode(null);
        }
    };

    const handleInstall = (code) => runAction(code, () => installPlugin(code), "messages.installSuccess", "messages.installError");
    const handleUpgrade = async (code) => {
        setUpgradeTarget(null);
        await runAction(code, () => upgradePlugin(code), "messages.upgradeSuccess", "messages.upgradeError");
    };
    const handleUninstall = async (code) => {
        setUninstallTarget(null);
        await runAction(code, () => uninstallPlugin(code), "messages.uninstallSuccess", "messages.uninstallError");
    };
    const handleToggleEnable = (code, enabled) => runAction(
        code,
        () => (enabled ? disablePlugin(code) : enablePlugin(code)),
        enabled ? "messages.disableSuccess" : "messages.enableSuccess",
        enabled ? "messages.disableError" : "messages.enableError",
    );
    const handleDelete = async (code) => {
        setDeleteTarget(null);
        await runAction(code, () => deletePlugin(code), "messages.deleteSuccess", "messages.deleteError");
    };

    const handleFile = async (file) => {
        if (!file.name.endsWith(".zip")) {
            setUploadError(t("upload.error.format"));
            return;
        }
        setUploadError("");
        setUploading(true);
        try {
            await uploadPlugin(file);
            setUploadOpen(false);
            await load();
        } catch (e) {
            setUploadError(e?.message || t("messages.uploadError"));
        } finally {
            setUploading(false);
            if (fileRef.current) fileRef.current.value = "";
        }
    };

    const handleDrag = (event) => {
        event.preventDefault();
        event.stopPropagation();
        if (event.type === "dragenter" || event.type === "dragover") setDragOver(true);
        else if (event.type === "dragleave") setDragOver(false);
    };

    const renderPluginCard = (plugin) => (
        <Card key={plugin.code} className="p-4">
            <div className="mb-2 flex items-center justify-between">
                <div className="flex min-w-0 flex-1 items-center gap-2">
                    <h3 className="truncate text-base font-semibold">{plugin.name}</h3>
                    {typeInfo(plugin.type) && (
                        <Badge variant="outline" className="border-primary/20 bg-primary/5 text-primary">{typeInfo(plugin.type).label}</Badge>
                    )}
                    {plugin.is_installed ? (
                        <Badge variant={plugin.is_enabled ? "default" : "outline"}>{plugin.is_enabled ? t("status.enabled") : t("status.disabled")}</Badge>
                    ) : (
                        <Badge variant="outline" className="border-muted-foreground/30 bg-muted/30 text-muted-foreground">{t("status.not_installed")}</Badge>
                    )}
                </div>
                <div className="flex shrink-0 items-center gap-1.5">
                    {plugin.is_protected && (
                        <span title={t("status.protected")} className="flex h-5 w-5 items-center justify-center rounded-full bg-amber-100 text-amber-600">!</span>
                    )}
                    {plugin.readme && (
                        <button
                            type="button"
                            title={t("button.readme")}
                            className="flex h-5 w-5 items-center justify-center rounded-full text-muted-foreground hover:bg-muted hover:text-primary"
                            onClick={() => { setReadmeContent(plugin.readme); setReadmeOpen(true); }}
                        >
                            ?
                        </button>
                    )}
                </div>
            </div>
            <div className="mb-2 flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
                <code className="rounded bg-muted px-1 py-0.5">{plugin.code}</code>
                <span>v{plugin.version}</span>
                <span>{t("author")}: {plugin.author}</span>
            </div>
            <p className="mb-3 line-clamp-2 text-sm text-muted-foreground">{plugin.description}</p>
            <div className="flex items-center justify-end gap-2">
                {plugin.is_installed ? (
                    <>
                        {plugin.need_upgrade && (
                            <Button variant="destructive" size="sm" disabled={actionCode === plugin.code} onClick={() => setUpgradeTarget(plugin)}>
                                {t("button.upgrade")}
                            </Button>
                        )}
                        {plugin.config && Object.keys(plugin.config).length > 0 && (
                            <Button variant="outline" size="sm" disabled={!plugin.is_enabled || actionCode === plugin.code} onClick={() => setConfigTarget(plugin)}>
                                {t("button.config")}
                            </Button>
                        )}
                        <Button
                            variant={plugin.is_enabled ? "destructive" : "default"}
                            size="sm"
                            disabled={actionCode === plugin.code}
                            onClick={() => handleToggleEnable(plugin.code, plugin.is_enabled)}
                        >
                            {plugin.is_enabled ? t("button.disable") : t("button.enable")}
                        </Button>
                        <Button
                            variant="outline"
                            size="sm"
                            className="text-muted-foreground hover:text-destructive"
                            disabled={actionCode === plugin.code || plugin.is_enabled}
                            onClick={() => setUninstallTarget(plugin)}
                        >
                            {t("button.uninstall")}
                        </Button>
                    </>
                ) : (
                    <>
                        <Button size="sm" disabled={actionCode === plugin.code} loading={actionCode === plugin.code} onClick={() => handleInstall(plugin.code)}>
                            {t("button.install")}
                        </Button>
                        {plugin.can_be_deleted !== false && (
                            <Button variant="ghost" size="sm" className="text-muted-foreground hover:text-destructive" disabled={actionCode === plugin.code} onClick={() => setDeleteTarget(plugin)}>
                                ×
                            </Button>
                        )}
                    </>
                )}
            </div>
        </Card>
    );

    return (
        <div className="p-4 md:p-6">
            <div className="flex flex-col gap-1">
                <h3 className="text-lg font-medium">{t("title")}</h3>
                <p className="text-sm text-muted-foreground">{t("description")}</p>
            </div>
            <hr className="my-6" />

            <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
                <Input
                    placeholder={t("search.placeholder")}
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    className="md:max-w-sm"
                />
                <div className="flex items-center gap-2">
                    <Select value={statusFilter} onValueChange={setStatusFilter} className="w-[140px]">
                        <SelectTrigger><SelectValue placeholder={t("status.filter_placeholder")} /></SelectTrigger>
                        <SelectContent>
                            <SelectItem value="all">{t("status.all")}</SelectItem>
                            <SelectItem value="installed">{t("status.installed")}</SelectItem>
                            <SelectItem value="available">{t("status.available")}</SelectItem>
                        </SelectContent>
                    </Select>
                    <Button variant="outline" size="sm" onClick={() => { setUploadError(""); setUploadOpen(true); }}>
                        {t("upload.button")}
                    </Button>
                </div>
            </div>

            <Tabs value={typeFilter} onValueChange={setTypeFilter} className="mt-6 w-full">
                <TabsList>
                    {types.map((tp) => <TabsTrigger key={tp.value} value={tp.value}>{tp.label}</TabsTrigger>)}
                    <TabsTrigger value="all">{t("tabs.all")}</TabsTrigger>
                </TabsList>
                <TabsContent value={typeFilter} className="mt-6">
                    <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                        {loading ? (
                            <div className="text-sm text-muted-foreground">...</div>
                        ) : filtered.length === 0 ? (
                            <div className="text-sm text-muted-foreground">-</div>
                        ) : (
                            filtered.map(renderPluginCard)
                        )}
                    </div>
                </TabsContent>
            </Tabs>

            <Dialog open={uploadOpen} onOpenChange={setUploadOpen} className="sm:max-w-md">
                <div className="p-6">
                    <h3 className="text-lg font-semibold">{t("upload.title")}</h3>
                    <p className="mt-1 text-sm text-muted-foreground">{t("upload.description")}</p>
                    <div
                        className={classNames(
                            "relative mt-4 flex h-64 flex-col items-center justify-center rounded-lg border-2 border-dashed border-muted-foreground/25 px-5 py-10 text-center transition-colors",
                            dragOver && "border-primary/50 bg-muted/50",
                        )}
                        onDragEnter={handleDrag}
                        onDragLeave={handleDrag}
                        onDragOver={handleDrag}
                        onDrop={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            setDragOver(false);
                            if (e.dataTransfer.files?.[0]) handleFile(e.dataTransfer.files[0]);
                        }}
                    >
                        <input
                            type="file"
                            ref={fileRef}
                            className="hidden"
                            accept=".zip"
                            onChange={(e) => e.target.files?.[0] && handleFile(e.target.files[0])}
                        />
                        {uploading ? (
                            <div className="flex flex-col items-center space-y-2">
                                <div className="h-10 w-10 animate-spin rounded-full border-b-2 border-primary" />
                                <div className="text-sm text-muted-foreground">{t("upload.uploading")}</div>
                            </div>
                        ) : (
                            <div className="flex flex-col items-center space-y-4">
                                <div className="text-sm font-medium">
                                    {t("upload.dragText")}{" "}
                                    <button type="button" onClick={() => fileRef.current?.click()} className="mx-1 text-primary hover:underline">
                                        {t("upload.clickText")}
                                    </button>
                                </div>
                                <div className="text-xs text-muted-foreground">{t("upload.supportText")}</div>
                            </div>
                        )}
                    </div>
                    {uploadError && <div className="mt-3 rounded-md bg-destructive/10 p-3 text-sm text-destructive">{uploadError}</div>}
                </div>
            </Dialog>

            <Dialog open={readmeOpen} onOpenChange={setReadmeOpen} className="sm:max-w-2xl">
                <div className="max-h-[80vh] overflow-y-auto p-6">
                    <h3 className="text-lg font-semibold">{t("readme.title")}</h3>
                    <pre className="mt-4 whitespace-pre-wrap font-mono text-xs text-muted-foreground">{readmeContent}</pre>
                </div>
            </Dialog>

            {configTarget && (
                <PluginConfigDialog
                    plugin={configTarget}
                    onOpenChange={(open) => !open && setConfigTarget(null)}
                    onSuccess={() => { setConfigTarget(null); load(); }}
                    t={t}
                />
            )}

            <ConfirmDialog
                open={Boolean(deleteTarget)}
                onOpenChange={(open) => !open && setDeleteTarget(null)}
                title={t("delete.title")}
                description={t("delete.description")}
                confirmLabel={t("delete.button")}
                onConfirm={() => handleDelete(deleteTarget.code)}
            />
            <ConfirmDialog
                open={Boolean(uninstallTarget)}
                onOpenChange={(open) => !open && setUninstallTarget(null)}
                title={t("uninstall.title")}
                description={t("uninstall.description")}
                confirmLabel={t("uninstall.button")}
                onConfirm={() => handleUninstall(uninstallTarget.code)}
            />
            <ConfirmDialog
                open={Boolean(upgradeTarget)}
                onOpenChange={(open) => !open && setUpgradeTarget(null)}
                title={t("upgrade.title")}
                description={t("upgrade.description")}
                confirmLabel={t("upgrade.button")}
                variant="default"
                onConfirm={() => handleUpgrade(upgradeTarget.code)}
            />
        </div>
    );
}

function PluginConfigDialog({ plugin, onOpenChange, onSuccess, t }) {
    const [loading, setLoading] = useState(true);
    const [schema, setSchema] = useState({});
    const [values, setValues] = useState({});
    const [error, setError] = useState("");

    useEffect(() => {
        setLoading(true);
        getPluginConfig(plugin.code)
            .then((res) => {
                const s = res?.data || {};
                setSchema(s);
                setValues(configSchemaToValues(s));
            })
            .catch(() => setError(t("messages.configLoadError")))
            .finally(() => setLoading(false));
    }, [plugin.code]);

    const handleSubmit = async () => {
        setError("");
        try {
            await updatePluginConfig(plugin.code, values);
            onSuccess?.();
        } catch (e) {
            setError(e?.message || t("messages.configSaveError"));
            throw e;
        }
    };

    return (
        <FormDialog
            open
            onOpenChange={onOpenChange}
            title={`${plugin.name} ${t("config.title")}`}
            description={t("config.description")}
            cancelLabel={t("config.cancel")}
            submitLabel={t("config.save")}
            onSubmit={handleSubmit}
            className="sm:max-w-lg"
        >
            {error && <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">{error}</div>}
            {loading ? (
                <div className="flex h-40 items-center justify-center text-sm text-muted-foreground">...</div>
            ) : (
                <DynamicConfigFields schema={schema} values={values} onChange={setValues} />
            )}
        </FormDialog>
    );
}
