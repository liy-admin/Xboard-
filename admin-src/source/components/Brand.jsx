import React from "react";

export const BRAND_NAME = "Lets VPN";

/**
 * Lets VPN 标志：渐变圆角方块 + 白色「L」+ 信号弧线。
 * 纯 SVG，随 size 缩放；渐变 id 带前缀避免与页面其他 SVG 冲突。
 */
export function BrandMark({ size = 36, className }) {
    return (
        <svg
            xmlns="http://www.w3.org/2000/svg"
            width={size}
            height={size}
            viewBox="0 0 48 48"
            role="img"
            aria-label={BRAND_NAME}
            className={className}
        >
            <defs>
                <linearGradient id="lv-mark-grad" x1="4" y1="4" x2="44" y2="44" gradientUnits="userSpaceOnUse">
                    <stop offset="0" stopColor="#2dd4bf" />
                    <stop offset="1" stopColor="#3b82f6" />
                </linearGradient>
            </defs>
            <rect x="2" y="2" width="44" height="44" rx="13" fill="url(#lv-mark-grad)" />
            <path d="M16 12v18a4 4 0 0 0 4 4h6" fill="none" stroke="#fff" strokeWidth="5" strokeLinecap="round" strokeLinejoin="round" />
            <path d="M29 22.5a8 8 0 0 1 8 8" fill="none" stroke="#fff" strokeOpacity=".9" strokeWidth="3.4" strokeLinecap="round" />
            <path d="M29 15.5a15 15 0 0 1 15 15" fill="none" stroke="#fff" strokeOpacity=".55" strokeWidth="3.4" strokeLinecap="round" />
        </svg>
    );
}

export function BrandLockup({ size = 36, className }) {
    return (
        <span className={["lv-brand", className].filter(Boolean).join(" ")}>
            <BrandMark size={size} />
            <span className="lv-brand__name">{BRAND_NAME}</span>
        </span>
    );
}
