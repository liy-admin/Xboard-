import React, { useEffect, useRef, useState } from "react";
import { useTranslation } from "../../i18n/index.js";
import { useAuthState } from "../../store/auth.js";
import { fetchThemes, getThemeConfig, saveThemeConfig, uploadTheme, deleteTheme } from "../../api/theme.js";
import { saveSettings } from "../../api/config.js";
import { Button } from "../../components/Button.jsx";
import { Input } from "../../components/Input.jsx";
import { Textarea } from "../../components/Textarea.jsx";
import { Card, CardHeader, CardTitle, CardContent, CardFooter } from "../../components/Card.jsx";
import { FormDialog } from "../../components/FormDialog.jsx";
import { ConfirmDialog } from "../../components/ConfirmDialog.jsx";
import { Dialog } from "../../components/Dialog.jsx";
import { classNames } from "../../utils/classNames.js";

export function ThemeConfig() {
    const { t } = useTranslation("theme");
    const auth = useAuthState();
    const [themes, setThemes] = useState({});
    const [active, setActive] = useState(null);
    const [loading, setLoading] = useState(true);
    const [activating, setActivating] = useState(null);
    const [deleting, setDeleting] = useState(null);
    const [deleteTarget, setDeleteTarget] = useState(null);
    const [deleteError, setDeleteError] = useState("");

    const [uploadOpen, setUploadOpen] = useState(false);
    const [uploading, setUploading] = useState(false);
    const [dragOver, setDragOver] = useState(false);
    const [uploadError, setUploadError] = useState("");
    const fileRef = useRef(null);

    const [configTarget, setConfigTarget] = useState(null);
    const [preview, setPreview] = useState(null);
    const [previewIndex, setPreviewIndex] = useState(0);

    const load = () => {
        setLoading(true);
        return fetchThemes()
            .then((res) => {
                setThemes(res?.data?.themes || {});
                setActive(res?.data?.active || null);
            })
            .finally(() => setLoading(false));
    };

    useEffect(() => {
        load();
    }, [auth.token]);

    const handleActivate = async (key) => {
        setActivating(key);
        try {
            await saveSettings({ frontend_theme: key });
            await load();
        } finally {
            setActivating(null);
        }
    };

    const handleDelete = async () => {
        if (!deleteTarget) return;
        setDeleteError("");
        if (deleteTarget === active) {
            setDeleteError(t("card.delete.error.active"));
            throw new Error(t("card.delete.error.active"));
        }
        setDeleting(deleteTarget);
        try {
            await deleteTheme(deleteTarget);
            await load();
        } finally {
            setDeleting(null);
        }
    };

    const handleFile = async (file) => {
        if (!file.name.endsWith(".zip")) {
            setUploadError(t("upload.error.format"));
            return;
        }
        setUploadError("");
        setUploading(true);
        try {
            await uploadTheme(file);
            setUploadOpen(false);
            await load();
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

    return (
        <div className="p-4 md:p-6">
            <div className="flex items-center justify-between">
                <div className="flex flex-col gap-1">
                    <h3 className="text-lg font-medium">{t("title")}</h3>
                    <p className="text-sm text-muted-foreground">{t("description")}</p>
                </div>
                <Button variant="outline" size="sm" onClick={() => { setUploadError(""); setUploadOpen(true); }}>
                    {t("upload.button")}
                </Button>
            </div>
            <hr className="my-6" />

            {loading ? (
                <div className="text-sm text-muted-foreground">...</div>
            ) : (
                <div className="grid gap-6 md:grid-cols-1 lg:grid-cols-2 xl:grid-cols-3">
                    {Object.entries(themes).map(([key, info]) => (
                        <Card
                            key={key}
                            className="group relative overflow-hidden"
                            style={info.background_url ? { backgroundImage: `url(${info.background_url})`, backgroundSize: "cover", backgroundPosition: "center" } : undefined}
                        >
                            <div className={classNames("relative z-10", info.background_url && "bg-background/85 backdrop-blur-[1px]")}>
                                {Boolean(info.can_delete) && (
                                    <div className="absolute right-2 top-2">
                                        <Button
                                            variant="ghost"
                                            size="icon"
                                            className="h-8 w-8 text-muted-foreground hover:text-destructive"
                                            disabled={deleting === key}
                                            loading={deleting === key}
                                            onClick={() => { setDeleteError(""); setDeleteTarget(key); }}
                                        >
                                            ×
                                        </Button>
                                    </div>
                                )}
                                <CardHeader>
                                    <CardTitle>{info.name}</CardTitle>
                                    <div className="space-y-2">
                                        {info.description && <div className="text-sm text-muted-foreground">{info.description}</div>}
                                        {info.version && <div className="text-sm text-muted-foreground">{t("card.version", { version: info.version })}</div>}
                                    </div>
                                </CardHeader>
                                <CardFooter className="flex items-center justify-end gap-3">
                                    {Array.isArray(info.images) && info.images.length > 0 && (
                                        <Button variant="outline" size="icon" className="h-8 w-8" onClick={() => { setPreview({ name: info.name, images: info.images }); setPreviewIndex(0); }}>
                                            🖼
                                        </Button>
                                    )}
                                    {Array.isArray(info.configs) && info.configs.length > 0 && (
                                        <Button variant="outline" onClick={() => setConfigTarget({ key, info })}>{t("card.configureTheme")}</Button>
                                    )}
                                    <Button
                                        onClick={() => handleActivate(key)}
                                        disabled={activating === key || key === active}
                                        loading={activating === key}
                                        variant={key === active ? "secondary" : "default"}
                                    >
                                        {key === active ? t("card.currentTheme") : t("card.activateTheme")}
                                    </Button>
                                </CardFooter>
                            </div>
                        </Card>
                    ))}
                </div>
            )}

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

            {configTarget && (
                <ThemeConfigDialog
                    themeKey={configTarget.key}
                    themeInfo={configTarget.info}
                    onOpenChange={(open) => !open && setConfigTarget(null)}
                    t={t}
                />
            )}

            <ConfirmDialog
                open={Boolean(deleteTarget)}
                onOpenChange={(open) => !open && setDeleteTarget(null)}
                title={t("card.delete.title")}
                description={deleteError || t("card.delete.description")}
                confirmLabel={t("card.delete.button")}
                onConfirm={handleDelete}
            />

            <Dialog open={Boolean(preview)} onOpenChange={(open) => !open && setPreview(null)} className="max-w-4xl">
                {preview && (
                    <div className="p-6">
                        <h3 className="text-lg font-semibold">{preview.name} {t("preview.title")}</h3>
                        <p className="mt-1 text-center text-sm text-muted-foreground">
                            {t("preview.imageCount", { current: previewIndex + 1, total: preview.images.length })}
                        </p>
                        <div className="relative mt-4">
                            <div className="aspect-video overflow-hidden rounded-lg border bg-muted">
                                <img src={preview.images[previewIndex]} alt="" className="h-full w-full object-contain" />
                            </div>
                            {preview.images.length > 1 && (
                                <>
                                    <Button variant="outline" size="icon" className="absolute left-4 top-1/2 h-8 w-8 -translate-y-1/2 rounded-full" onClick={() => setPreviewIndex((i) => (i === 0 ? preview.images.length - 1 : i - 1))}>‹</Button>
                                    <Button variant="outline" size="icon" className="absolute right-4 top-1/2 h-8 w-8 -translate-y-1/2 rounded-full" onClick={() => setPreviewIndex((i) => (i === preview.images.length - 1 ? 0 : i + 1))}>›</Button>
                                </>
                            )}
                        </div>
                    </div>
                )}
            </Dialog>
        </div>
    );
}

function ThemeConfigDialog({ themeKey, themeInfo, onOpenChange, t }) {
    const [loading, setLoading] = useState(true);
    const [values, setValues] = useState({});

    useEffect(() => {
        setLoading(true);
        getThemeConfig(themeKey)
            .then((res) => setValues(res?.data || {}))
            .finally(() => setLoading(false));
    }, [themeKey]);

    const handleSubmit = async () => {
        await saveThemeConfig(themeKey, values);
    };

    return (
        <FormDialog
            open
            onOpenChange={onOpenChange}
            title={t("config.title", { name: themeInfo.name })}
            description={t("config.description")}
            cancelLabel={t("config.cancel")}
            submitLabel={t("config.save")}
            onSubmit={handleSubmit}
            className="sm:max-w-md"
        >
            {loading ? (
                <div className="flex h-40 items-center justify-center text-sm text-muted-foreground">...</div>
            ) : (
                (themeInfo.configs || []).map((field) => (
                    <div key={field.field_name} className="space-y-1.5">
                        <div className="text-sm font-medium">{field.label}</div>
                        {field.field_type === "textarea" ? (
                            <Textarea
                                placeholder={field.placeholder}
                                value={values[field.field_name] ?? ""}
                                onChange={(e) => setValues((p) => ({ ...p, [field.field_name]: e.target.value }))}
                            />
                        ) : field.field_type === "select" ? (
                            <select
                                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm"
                                value={values[field.field_name] ?? ""}
                                onChange={(e) => setValues((p) => ({ ...p, [field.field_name]: e.target.value }))}
                            >
                                {Object.entries(field.select_options || {}).map(([value, label]) => (
                                    <option key={value} value={value}>{label}</option>
                                ))}
                            </select>
                        ) : (
                            <Input
                                placeholder={field.placeholder}
                                value={values[field.field_name] ?? ""}
                                onChange={(e) => setValues((p) => ({ ...p, [field.field_name]: e.target.value }))}
                            />
                        )}
                    </div>
                ))
            )}
        </FormDialog>
    );
}
