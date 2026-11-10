import React, { useMemo, useState } from "react";
import { TableHead } from "./Table.jsx";

// 表头点击排序：同一列依次 升序 → 降序 → 取消。
// 整表已在前端的列表用 useClientSort 在本地排；分页列表用 useServerSort 把 sort 交给后端。

function nextSort(prev, key) {
    if (prev.key !== key) return { key, dir: "asc" };
    if (prev.dir === "asc") return { key, dir: "desc" };
    return { key: null, dir: "asc" };
}

function isNullish(v) {
    return v === null || v === undefined || v === "" || (typeof v === "number" && !Number.isFinite(v));
}

function compareValues(va, vb) {
    if (typeof va === "string" && typeof vb === "string") return va.localeCompare(vb, undefined, { numeric: true, sensitivity: "base" });
    return va < vb ? -1 : va > vb ? 1 : 0;
}

// getters: { 列键: (row) => 用于比较的值 }；空值无论升降序都排在最后。
export function useClientSort(rows, getters, initial = { key: null, dir: "asc" }) {
    const [sort, setSort] = useState(initial);
    const sorted = useMemo(() => {
        const get = sort.key ? getters[sort.key] : null;
        if (!get || !Array.isArray(rows)) return rows;
        const arr = [...rows];
        arr.sort((a, b) => {
            const va = get(a);
            const vb = get(b);
            const an = isNullish(va);
            const bn = isNullish(vb);
            if (an && bn) return 0;
            if (an) return 1;
            if (bn) return -1;
            const cmp = compareValues(va, vb);
            return sort.dir === "asc" ? cmp : -cmp;
        });
        return arr;
        // getters 每次渲染都是新对象，排序只依赖数据与当前排序列
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [rows, sort]);
    return { sorted, sort, toggle: (key) => setSort((p) => nextSort(p, key)) };
}

// 分页列表：onChange 用于把页码重置到第一页。sortParam 直接作为请求参数 sort 传给后端。
export function useServerSort(onChange) {
    const [sort, setSort] = useState({ key: null, dir: "asc" });
    const toggle = (key) => {
        setSort((p) => nextSort(p, key));
        onChange?.();
    };
    const sortParam = sort.key ? [{ id: sort.key, desc: sort.dir === "desc" }] : [];
    return { sort, toggle, sortParam };
}

// SortLabel 是表头里可点击的标题；用于手写 <th> 的小表格。
export function SortLabel({ sortKey, sort, onSort, children }) {
    const active = sort?.key === sortKey;
    return (
        <button type="button" onClick={() => onSort(sortKey)} className="inline-flex items-center gap-1 select-none hover:text-foreground">
            {children}
            <span className={active ? "text-[10px] text-foreground" : "text-[10px] text-muted-foreground/60"}>
                {active ? (sort.dir === "asc" ? "▲" : "▼") : "↕"}
            </span>
        </button>
    );
}

export function SortableHead({ sortKey, sort, onSort, className, disabled, children, ...rest }) {
    if (disabled || !onSort) return <TableHead className={className} {...rest}>{children}</TableHead>;
    const active = sort?.key === sortKey;
    return (
        <TableHead className={className} {...rest} aria-sort={active ? (sort.dir === "asc" ? "ascending" : "descending") : "none"}>
            <SortLabel sortKey={sortKey} sort={sort} onSort={onSort}>{children}</SortLabel>
        </TableHead>
    );
}
