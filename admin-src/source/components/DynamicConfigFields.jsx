import React from "react";
import { Input } from "./Input.jsx";
import { Textarea } from "./Textarea.jsx";
import { Switch } from "./Switch.jsx";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "./Select.jsx";

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
                    default:
                        return null;
                }
            })}
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
