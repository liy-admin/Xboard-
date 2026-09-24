import React from "react";
import { classNames } from "../../utils/classNames.js";
import { buildHashHref } from "../../utils/query.js";
import { useHashLocation } from "../../composables/useHashLocation.js";
import { useTranslation } from "../../i18n/index.js";

const SIDEBAR_ITEMS = [
    { key: "site", href: "/config/system", icon: "settings" },
    { key: "safe", href: "/config/system/safe", icon: "shield" },
    { key: "subscribe", href: "/config/system/subscribe", icon: "ticket" },
    { key: "invite", href: "/config/system/invite", icon: "users" },
    { key: "lucky_draw", href: "/config/system/lucky-draw", icon: "gift" },
    { key: "discover", href: "/config/system/discover", icon: "flame" },
    { key: "ad_banner", href: "/config/system/ad-banner", icon: "photo" },
    { key: "support", href: "/config/system/support", icon: "headset" },
    { key: "server", href: "/config/system/server", icon: "server" },
    { key: "email", href: "/config/system/email", icon: "mail" },
    { key: "telegram", href: "/config/system/telegram", icon: "send" },
    { key: "app", href: "/config/system/app", icon: "smartphone" },
    { key: "subscribe_template", href: "/config/system/subscribe-template", icon: "code" },
];

function SettingsIcon({ name, size = 18 }) {
    const icons = {
        settings: <><path d="M10.325 4.317c.426 -1.756 2.924 -1.756 3.35 0a1.724 1.724 0 0 0 2.573 1.066c1.543 -.94 3.31 .826 2.37 2.37a1.724 1.724 0 0 0 1.065 2.572c1.756 .426 1.756 2.924 0 3.35a1.724 1.724 0 0 0 -1.066 2.573c.94 1.543 -.826 3.31 -2.37 2.37a1.724 1.724 0 0 0 -2.572 1.065c-.426 1.756 -2.924 1.756 -3.35 0a1.724 1.724 0 0 0 -2.573 -1.066c-1.543 .94 -3.31 -.826 -2.37 -2.37a1.724 1.724 0 0 0 -1.065 -2.572c-1.756 -.426 -1.756 -2.924 0 -3.35a1.724 1.724 0 0 0 1.066 -2.573c-.94 -1.543 .826 -3.31 2.37 -2.37c1 .608 2.296 .07 2.572 -1.065z" /><path d="M9 12a3 3 0 1 0 6 0a3 3 0 0 0 -6 0" /></>,
        shield: <><path d="M12 3a12 12 0 0 0 8.5 3a12 12 0 0 1 -8.5 15a12 12 0 0 1 -8.5 -15a12 12 0 0 0 8.5 -3" /></>,
        ticket: <><path d="M15 5l0 2" /><path d="M15 11l0 2" /><path d="M15 17l0 2" /><path d="M5 5h14a2 2 0 0 1 2 2v3a2 2 0 0 0 0 4v3a2 2 0 0 1 -2 2h-14a2 2 0 0 1 -2 -2v-3a2 2 0 0 0 0 -4v-3a2 2 0 0 1 2 -2" /></>,
        users: <><path d="M9 7m-4 0a4 4 0 1 0 8 0a4 4 0 1 0 -8 0" /><path d="M3 21v-2a4 4 0 0 1 4 -4h4a4 4 0 0 1 4 4v2" /><path d="M16 3.13a4 4 0 0 1 0 7.75" /><path d="M21 21v-2a4 4 0 0 0 -3 -3.85" /></>,
        server: <><path d="M3 4m0 3a3 3 0 0 1 3 -3h12a3 3 0 0 1 3 3v2a3 3 0 0 1 -3 3h-12a3 3 0 0 1 -3 -3z" /><path d="M3 12m0 3a3 3 0 0 1 3 -3h12a3 3 0 0 1 3 3v2a3 3 0 0 1 -3 3h-12a3 3 0 0 1 -3 -3z" /><path d="M7 8l0 .01" /><path d="M7 16l0 .01" /></>,
        mail: <><rect x="3" y="5" width="18" height="14" rx="2" /><path d="M3 7l9 6l9 -6" /></>,
        send: <><path d="M10 14l11 -11" /><path d="M21 3l -6.5 18a.55 .55 0 0 1 -1 0l -3.5 -7l -7 -3.5a.55 .55 0 0 1 0 -1z" /></>,
        smartphone: <><rect x="7" y="4" width="10" height="16" rx="1" /><path d="M11 5h2" /><path d="M12 17v.01" /></>,
        code: <><path d="M7 8l -4 4l4 4" /><path d="M17 8l4 4l -4 4" /></>,
        gift: <><rect x="3" y="8" width="18" height="4" rx="1" /><path d="M12 8l0 13" /><path d="M19 12v7a2 2 0 0 1 -2 2h-10a2 2 0 0 1 -2 -2v-7" /><path d="M7.5 8a2.5 2.5 0 0 1 0 -5c1.5 0 3 1.5 4.5 5c1.5 -3.5 3 -5 4.5 -5a2.5 2.5 0 0 1 0 5" /></>,
        flame: <><path d="M12 12c2 -2.96 0 -7 -1 -8c0 3.038 -1.773 4.741 -3 6c-1.226 1.26 -2 3.24 -2 5c0 3.314 2.686 6 6 6s6 -2.686 6 -6c0 -1.26 -.472 -2.86 -1 -4c-1.5 3.5 -3 4.375 -4.999 4.999c0 .001 0 .001 0 .001z" /></>,
        headset: <><path d="M4 14v-3a8 8 0 1 1 16 0v3" /><path d="M18 19c0 1.657 -2.686 3 -6 3" /><path d="M4 14a2 2 0 0 1 2 -2h1a1 1 0 0 1 1 1v5a1 1 0 0 1 -1 1h-1a2 2 0 0 1 -2 -2z" /><path d="M16 14a2 2 0 0 1 2 -2h1a2 2 0 0 1 2 2v3a2 2 0 0 1 -2 2h-1a1 1 0 0 1 -1 -1v-5a1 1 0 0 1 1 -1" /></>,
        photo: <><path d="M15 8h.01" /><path d="M3 6a3 3 0 0 1 3 -3h12a3 3 0 0 1 3 3v12a3 3 0 0 1 -3 3h-12a3 3 0 0 1 -3 -3z" /><path d="M3 16l5 -5c.928 -.893 2.072 -.893 3 0l5 5" /><path d="M14 14l1 -1c.928 -.893 2.072 -.893 3 0l3 3" /></>,
    };

    return (
        <svg xmlns="http://www.w3.org/2000/svg" width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="settings-nav-icon">
            {icons[name] || null}
        </svg>
    );
}

function SidebarNav({ items, currentPath }) {
    const { t } = useTranslation("settings");

    return (
        <nav className="flex space-x-2 lg:flex-col lg:space-x-0 lg:space-y-1">
            {items.map((item) => (
                <a
                    key={item.key}
                    href={buildHashHref(item.href)}
                    className={classNames(
                        "inline-flex items-center justify-start gap-2 whitespace-nowrap rounded-md px-3 py-2 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground",
                        currentPath === item.href
                            ? "bg-muted hover:bg-muted"
                            : "hover:bg-transparent hover:underline",
                    )}
                >
                    <SettingsIcon name={item.icon} />
                    {item.title || t(item.key + ".title")}
                </a>
            ))}
        </nav>
    );
}

export function SettingsSidebarContent({ items = SIDEBAR_ITEMS, currentPath }) {
    const { t } = useTranslation("settings");
    return (
        <>
            <div className="p-1 md:hidden">
                <select
                    className="h-12 w-full rounded-md border border-input bg-background px-3 text-sm sm:w-48"
                    value={currentPath}
                    onChange={(e) => {
                        window.location.hash = e.target.value.startsWith("/") ? e.target.value : `/${e.target.value}`;
                    }}
                >
                    {items.map((item) => (
                        <option key={item.href} value={item.href}>
                            {item.title || t(item.key + ".title")}
                        </option>
                    ))}
                </select>
            </div>
            <div className="hidden w-full overflow-x-auto bg-background px-1 py-2 md:block">
                <SidebarNav items={items} currentPath={currentPath} />
            </div>
        </>
    );
}
