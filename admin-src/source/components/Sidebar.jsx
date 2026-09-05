import React, { useEffect, useMemo, useRef, useState } from "react";
import { buildHashHref } from "../utils/query.js";
import { useHashLocation } from "../composables/useHashLocation.js";
import { classNames } from "../utils/classNames.js";
import { useTranslation } from "../i18n/index.js";

const ICONS = {
    logo: <><line x1="208" y1="128" x2="128" y2="208" /><line x1="192" y1="40" x2="40" y2="192" /></>,
    dashboard: <><path d="M12 13m-2 0a2 2 0 1 0 4 0a2 2 0 1 0 -4 0" /><path d="M13.45 11.55l2.05 -2.05" /><path d="M6.4 20a9 9 0 1 1 11.2 0z" /></>,
    settings: <><path d="M10.325 4.317c.426 -1.756 2.924 -1.756 3.35 0a1.724 1.724 0 0 0 2.573 1.066c1.543 -.94 3.31 .826 2.37 2.37a1.724 1.724 0 0 0 1.065 2.572c1.756 .426 1.756 2.924 0 3.35a1.724 1.724 0 0 0 -1.066 2.573c.94 1.543 -.826 3.31 -2.37 2.37a1.724 1.724 0 0 0 -2.572 1.065c-.426 1.756 -2.924 1.756 -3.35 0a1.724 1.724 0 0 0 -2.573 -1.066c-1.543 .94 -3.31 -.826 -2.37 -2.37a1.724 1.724 0 0 0 -1.065 -2.572c-1.756 -.426 -1.756 -2.924 0 -3.35a1.724 1.724 0 0 0 1.066 -2.573c-.94 -1.543 .826 -3.31 2.37 -2.37c1 .608 2.296 .07 2.572 -1.065z" /><path d="M9 12a3 3 0 1 0 6 0a3 3 0 0 0 -6 0" /></>,
    adjustments: <><path d="M4 10a2 2 0 1 0 4 0a2 2 0 0 0 -4 0" /><path d="M6 4v4" /><path d="M6 12v8" /><path d="M10 16a2 2 0 1 0 4 0a2 2 0 0 0 -4 0" /><path d="M12 4v10" /><path d="M12 18v2" /><path d="M16 7a2 2 0 1 0 4 0a2 2 0 0 0 -4 0" /><path d="M18 4v1" /><path d="M18 9v11" /></>,
    package: <><path d="m7.5 4.27 9 5.15" /><path d="M21 8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16Z" /><path d="m3.3 7 8.7 5 8.7-5" /><path d="M12 22V12" /></>,
    desktop: <><path d="M3 5a1 1 0 0 1 1 -1h16a1 1 0 0 1 1 1v10a1 1 0 0 1 -1 1h-16a1 1 0 0 1 -1 -1v-10z" /><path d="M7 20h10" /><path d="M9 16v4" /><path d="M15 16v4" /></>,
    news: <><path d="M16 6h3a1 1 0 0 1 1 1v11a2 2 0 0 1 -4 0v-13a1 1 0 0 0 -1 -1h-10a1 1 0 0 0 -1 1v12a3 3 0 0 0 3 3h11" /><path d="M8 8l4 0" /><path d="M8 12l4 0" /><path d="M8 16l4 0" /></>,
    card: <><path d="M3 5m0 3a3 3 0 0 1 3 -3h12a3 3 0 0 1 3 3v8a3 3 0 0 1 -3 3h-12a3 3 0 0 1 -3 -3z" /><path d="M3 10l18 0" /><path d="M7 15l.01 0" /><path d="M11 15l2 0" /></>,
    file: <><path d="M14 3v4a1 1 0 0 0 1 1h4" /><path d="M17 21h-10a2 2 0 0 1 -2 -2v-14a2 2 0 0 1 2 -2h7l5 5v11a2 2 0 0 1 -2 2z" /><path d="M9 9l1 0" /><path d="M9 13l6 0" /><path d="M9 17l6 0" /></>,
    server: <><path d="M3 4m0 3a3 3 0 0 1 3 -3h12a3 3 0 0 1 3 3v2a3 3 0 0 1 -3 3h-12a3 3 0 0 1 -3 -3z" /><path d="M3 12m0 3a3 3 0 0 1 3 -3h12a3 3 0 0 1 3 3v2a3 3 0 0 1 -3 3h-12a3 3 0 0 1 -3 -3z" /><path d="M7 8l0 .01" /><path d="M7 16l0 .01" /></>,
    lock: <><path d="M5 13a2 2 0 0 1 2 -2h10a2 2 0 0 1 2 2v6a2 2 0 0 1 -2 2h-10a2 2 0 0 1 -2 -2v-6z" /><path d="M11 16a1 1 0 1 0 2 0a1 1 0 0 0 -2 0" /><path d="M8 11v-4a4 4 0 1 1 8 0v4" /></>,
    route: <><path d="M3 19a2 2 0 1 0 4 0a2 2 0 0 0 -4 0" /><path d="M19 7a2 2 0 1 0 0 -4a2 2 0 0 0 0 4z" /><path d="M11 19h5.5a3.5 3.5 0 0 0 0 -7h-8a3.5 3.5 0 0 1 0 -7h4.5" /></>,
    cash: <><path d="M7 9m0 2a2 2 0 0 1 2 -2h10a2 2 0 0 1 2 2v6a2 2 0 0 1 -2 2h-10a2 2 0 0 1 -2 -2z" /><path d="M14 14m-2 0a2 2 0 1 0 4 0a2 2 0 1 0 -4 0" /><path d="M17 9v-2a2 2 0 0 0 -2 -2h-10a2 2 0 0 0 -2 2v6a2 2 0 0 0 2 2h2" /></>,
    store: <><path d="M3 21l18 0" /><path d="M3 7v1a3 3 0 0 0 6 0v-1m0 1a3 3 0 0 0 6 0v-1m0 1a3 3 0 0 0 6 0v-1h-18l2 -4h14l2 4" /><path d="M5 21l0 -10.15" /><path d="M19 21l0 -10.15" /><path d="M9 21v-4a2 2 0 0 1 2 -2h2a2 2 0 0 1 2 2v4" /></>,
    discount: <><path d="M5 7.2a2.2 2.2 0 0 1 2.2 -2.2h1a2.2 2.2 0 0 0 1.55 -.64l.7 -.7a2.2 2.2 0 0 1 3.12 0l.7 .7c.412 .41 .97 .64 1.55 .64h1a2.2 2.2 0 0 1 2.2 2.2v1c0 .58 .23 1.138 .64 1.55l.7 .7a2.2 2.2 0 0 1 0 3.12l-.7 .7a2.2 2.2 0 0 0 -.64 1.55v1a2.2 2.2 0 0 1 -2.2 2.2h-1a2.2 2.2 0 0 0 -1.55 .64l-.7 .7a2.2 2.2 0 0 1 -3.12 0l-.7 -.7a2.2 2.2 0 0 0 -1.55 -.64h-1a2.2 2.2 0 0 1 -2.2 -2.2v-1a2.2 2.2 0 0 0 -.64 -1.55l-.7 -.7a2.2 2.2 0 0 1 0 -3.12l.7 -.7a2.2 2.2 0 0 0 .64 -1.55v-1" /><path d="M9 12l2 2l4 -4" /></>,
    gift: <><path d="M3 8m0 1a1 1 0 0 1 1 -1h16a1 1 0 0 1 1 1v2a1 1 0 0 1 -1 1h-16a1 1 0 0 1 -1 -1z" /><path d="M12 8l0 13" /><path d="M19 12v7a2 2 0 0 1 -2 2h-10a2 2 0 0 1 -2 -2v-7" /><path d="M7.5 8a2.5 2.5 0 0 1 0 -5a4.8 8 0 0 1 4.5 5a4.8 8 0 0 1 4.5 -5a2.5 2.5 0 0 1 0 5" /></>,
    users: <><path d="M9 7m-4 0a4 4 0 1 0 8 0a4 4 0 1 0 -8 0" /><path d="M3 21v-2a4 4 0 0 1 4 -4h4a4 4 0 0 1 4 4v2" /><path d="M16 3.13a4 4 0 0 1 0 7.75" /><path d="M21 21v-2a4 4 0 0 0 -3 -3.85" /></>,
    user: <><path d="M8 7a4 4 0 1 0 8 0a4 4 0 0 0 -8 0" /><path d="M6 21v-2a4 4 0 0 1 4 -4h4a4 4 0 0 1 4 4v2" /></>,
    ticket: <><path d="M15 5l0 2" /><path d="M15 11l0 2" /><path d="M15 17l0 2" /><path d="M5 5h14a2 2 0 0 1 2 2v3a2 2 0 0 0 0 4v3a2 2 0 0 1 -2 2h-14a2 2 0 0 1 -2 -2v-3a2 2 0 0 0 0 -4v-3a2 2 0 0 1 2 -2" /></>,
    chevron: <path d="M6 9l6 6l6 -6" />,
    x: <><path d="M18 6l-12 12" /><path d="M6 6l12 12" /></>,
    menu: <><path d="M4 6h16" /><path d="M4 12h16" /><path d="M4 18h16" /></>,
};

const NAV_GROUPS = [
    { id: "dashboard", items: [{ titleKey: "dashboard", path: "/", icon: "dashboard" }] },
    { id: "system", titleKey: "systemManagement", icon: "settings", items: [
        { titleKey: "systemConfig", path: "/config/system", icon: "adjustments" },
        { titleKey: "pluginManagement", path: "/config/plugin", icon: "package" },
        { titleKey: "themeConfig", path: "/config/theme", icon: "desktop" },
        { titleKey: "noticeManagement", path: "/config/notice", icon: "news" },
        { titleKey: "paymentConfig", path: "/config/payment", icon: "card" },
        { titleKey: "knowledgeManagement", path: "/config/knowledge", icon: "file" },
    ] },
    { id: "node", titleKey: "nodeManagement", icon: "server", items: [
        { titleKey: "machineManagement", path: "/server/machine", icon: "server" },
        { titleKey: "nodeManagement", path: "/server/manage", icon: "server" },
        { titleKey: "permissionGroupManagement", path: "/server/group", icon: "lock" },
        { titleKey: "routeManagement", path: "/server/route", icon: "route" },
    ] },
    { id: "subscription", titleKey: "subscriptionManagement", icon: "cash", items: [
        { titleKey: "planManagement", path: "/finance/plan", icon: "store" },
        { titleKey: "orderManagement", path: "/finance/order", icon: "card" },
        { titleKey: "couponManagement", path: "/finance/coupon", icon: "discount" },
        { titleKey: "giftCardManagement", path: "/finance/gift-card", icon: "gift" },
    ] },
    { id: "user", titleKey: "userManagement", icon: "users", items: [
        { titleKey: "userManagement", path: "/user/manage", icon: "user" },
        { titleKey: "ticketManagement", path: "/user/ticket", icon: "ticket" },
    ] },
];

function Icon({ name, size = 18, strokeWidth = 2, className }) {
    const isLogo = name === "logo";
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width={isLogo ? undefined : size} height={isLogo ? undefined : size} viewBox={isLogo ? "0 0 256 256" : "0 0 24 24"} fill="none" stroke="currentColor" strokeWidth={strokeWidth} strokeLinecap="round" strokeLinejoin="round" className={className}>
            {isLogo && <rect width="256" height="256" fill="none" stroke="none" />}
            {ICONS[name]}
        </svg>
    );
}

export { Icon as SidebarIcon, NAV_GROUPS as SIDEBAR_NAV_GROUPS };

function NavLink({ item, currentPath, onNavigate }) {
    const active = currentPath === item.path;
    return (
        <a className={classNames("inline-flex items-center whitespace-nowrap font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 text-xs justify-start text-wrap rounded-none", active ? "bg-secondary text-secondary-foreground shadow-sm hover:bg-secondary/80" : "hover:bg-accent hover:text-accent-foreground", "h-10 w-full border-l border-l-slate-500 px-2")} href={buildHashHref(item.path)} aria-current={active ? "page" : undefined} onClick={onNavigate}>
            <div className="mr-2"><Icon name={item.icon} /></div>
            <span className="admin-sidebar-label">{item.title}</span>
        </a>
    );
}

function CollapsedNavLink({ item, currentPath, onNavigate }) {
    const active = currentPath === item.path;

    return (
        <div className="admin-sidebar-collapsed-item">
            <a className={classNames("admin-sidebar-collapsed-trigger inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 h-12 w-12", active ? "bg-secondary text-secondary-foreground shadow-sm hover:bg-secondary/80" : "hover:bg-accent hover:text-accent-foreground")} href={buildHashHref(item.path)} aria-current={active ? "page" : undefined} onClick={onNavigate}>
                <Icon name={item.icon} />
                <span className="sr-only">{item.title}</span>
            </a>
            <div className="admin-sidebar-collapsed-tooltip" role="tooltip">
                <span>{item.title}</span>
            </div>
        </div>
    );
}

function CollapsedNavGroup({ group, currentPath, onNavigate }) {
    const [open, setOpen] = useState(false);
    const rootRef = useRef(null);
    const active = group.items.some((item) => item.path === currentPath);

    useEffect(() => {
        if (!open) {
            return undefined;
        }

        const handlePointerDown = (event) => {
            if (!rootRef.current?.contains(event.target)) {
                setOpen(false);
            }
        };
        const handleKeyDown = (event) => {
            if (event.key === "Escape") {
                setOpen(false);
            }
        };

        document.addEventListener("pointerdown", handlePointerDown);
        document.addEventListener("keydown", handleKeyDown);
        return () => {
            document.removeEventListener("pointerdown", handlePointerDown);
            document.removeEventListener("keydown", handleKeyDown);
        };
    }, [open]);

    const handleNavigate = () => {
        setOpen(false);
        onNavigate?.();
    };

    return (
        <div ref={rootRef} className={classNames("admin-sidebar-collapsed-item", open && "admin-sidebar-collapsed-item-open")}>
            <button type="button" className={classNames("admin-sidebar-collapsed-trigger inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 h-12 w-12", active ? "bg-secondary text-secondary-foreground shadow-sm hover:bg-secondary/80" : "hover:bg-accent hover:text-accent-foreground")} aria-haspopup="menu" aria-expanded={open} onClick={() => setOpen((value) => !value)}>
                <Icon name={group.icon} />
                <span className="sr-only">{group.title}</span>
            </button>
            <div className="admin-sidebar-collapsed-tooltip" role="tooltip">
                <span>{group.title}</span>
                <Icon name="chevron" size={18} strokeWidth={1} className="admin-sidebar-collapsed-tooltip-chevron" />
            </div>
            {open && (
                <div className="admin-sidebar-collapsed-menu" role="menu">
                    <div className="admin-sidebar-collapsed-menu-label">{group.title}</div>
                    <div className="admin-sidebar-collapsed-menu-separator" />
                    {group.items.map((item) => {
                        const itemActive = currentPath === item.path;
                        return (
                            <a className={classNames("admin-sidebar-collapsed-menu-item", itemActive && "bg-secondary")} href={buildHashHref(item.path)} role="menuitem" aria-current={itemActive ? "page" : undefined} onClick={handleNavigate} key={item.path}>
                                <Icon name={item.icon} />
                                <span className="ml-2 max-w-52 text-wrap">{item.title}</span>
                            </a>
                        );
                    })}
                </div>
            )}
        </div>
    );
}

function NavGroup({ group, currentPath, onNavigate, collapsed }) {
    const [open, setOpen] = useState(true);
    const active = group.items.some((item) => item.path === currentPath);

    if (!group.title) {
        const item = group.items[0];
        if (collapsed) {
            return <CollapsedNavLink item={item} currentPath={currentPath} onNavigate={onNavigate} />;
        }
        return (
            <a className={classNames("inline-flex items-center whitespace-nowrap font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 text-xs h-12 justify-start text-wrap rounded-none px-6", currentPath === item.path ? "bg-secondary text-secondary-foreground shadow-sm hover:bg-secondary/80" : "hover:bg-accent hover:text-accent-foreground")} href={buildHashHref(item.path)} aria-current={currentPath === item.path ? "page" : undefined} onClick={onNavigate}>
                <div className="mr-2"><Icon name={item.icon} /></div>
                <span className="admin-sidebar-label">{item.title}</span>
            </a>
        );
    }

    if (collapsed) {
        return <CollapsedNavGroup group={group} currentPath={currentPath} onNavigate={onNavigate} />;
    }

    return (
        <div data-state={open ? "open" : "closed"}>
            <button type="button" aria-expanded={open} data-state={open ? "open" : "closed"} onClick={() => setOpen((value) => !value)} className={classNames("inline-flex items-center whitespace-nowrap font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 text-xs group h-12 w-full justify-start rounded-none px-6", active ? "bg-secondary text-secondary-foreground shadow-sm hover:bg-secondary/80" : "hover:bg-accent hover:text-accent-foreground")}>
                <div className="mr-2"><Icon name={group.icon} /></div>
                <span className="admin-sidebar-label">{group.title}</span>
                <span className={classNames("admin-sidebar-chevron ml-auto transition-all", open && "-rotate-180")}><Icon name="chevron" strokeWidth={1} /></span>
            </button>
            {open && (
                <ul data-state="open" className="collapsibleDropdown">
                    {group.items.map((item) => (
                        <li className="my-1 ml-8" key={item.path}><NavLink item={item} currentPath={currentPath} onNavigate={onNavigate} /></li>
                    ))}
                </ul>
            )}
        </div>
    );
}

export function Sidebar({ collapsed = false, onToggleCollapsed, onMobileOpenChange }) {
    const { t } = useTranslation("nav");
    const { t: tCommon } = useTranslation("common");
    const location = useHashLocation();
    const [mobileOpen, setMobileOpen] = useState(false);
    const version = useMemo(() => `v${new Date().toISOString().slice(0, 10).replace(/-/g, "")}`, []);
    const navGroups = useMemo(() => NAV_GROUPS.map((group) => ({
        ...group,
        title: group.titleKey ? t(group.titleKey) : undefined,
        items: group.items.map((item) => ({
            ...item,
            title: t(item.titleKey),
        })),
    })), [t]);

    useEffect(() => {
        onMobileOpenChange?.(mobileOpen);
    }, [mobileOpen, onMobileOpenChange]);

    return (
        <aside className={classNames("admin-sidebar fixed left-0 right-0 top-0 z-50 flex h-auto flex-col border-r-2 border-r-muted transition-[width] md:bottom-0 md:right-auto md:h-svh", collapsed ? "admin-sidebar-collapsed" : "admin-sidebar-expanded", mobileOpen && "admin-sidebar-mobile-open")} data-collapsed={collapsed} data-mobile-open={mobileOpen}>
            <div className="relative w-full flex flex-col h-[var(--header-height)] md:h-full bg-background">
                <div className="flex h-[var(--header-height)] flex-none items-center gap-4 bg-background p-4 sticky top-0 justify-between px-4 py-3 shadow md:px-4">
                    <div className="flex items-center gap-2">
                        <Icon name="logo" className="admin-sidebar-logo transition-all h-8 w-8" />
                        <span className="sr-only">Website Name</span>
                        <div className="admin-sidebar-label flex flex-col justify-end truncate visible w-auto"><span className="font-medium">XBoard</span></div>
                    </div>
                    <button className="inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 hover:bg-accent hover:text-accent-foreground h-9 w-9 md:hidden" aria-label={tCommon("toggleNavigation")} aria-controls="sidebar-menu" aria-expanded={mobileOpen} onClick={() => setMobileOpen((value) => !value)}>
                        <Icon name={mobileOpen ? "x" : "menu"} size={24} />
                    </button>
                </div>
                <div data-collapsed={collapsed} className="group border-b bg-background py-2 transition-[max-height,padding] duration-500 data-[collapsed=true]:py-2 md:border-none flex-1 overflow-auto overscroll-contain block md:py-2" id="sidebar-menu">
                    <nav className="grid gap-1 group-[[data-collapsed=true]]:justify-center group-[[data-collapsed=true]]:px-2">
                        {navGroups.map((group, index) => <NavGroup group={group} currentPath={location.pathname} onNavigate={() => setMobileOpen(false)} collapsed={collapsed} key={group.id || index} />)}
                    </nav>
                </div>
                <div className="admin-sidebar-footer border-t border-border/50 bg-background px-4 py-2.5 text-xs text-muted-foreground block text-left">
                    <div className="flex items-center gap-1.5 justify-start"><div className="admin-sidebar-footer-dot h-1.5 w-1.5 rounded-full bg-green-500" /><span className="admin-sidebar-footer-label whitespace-nowrap tracking-wide transition-opacity duration-200">{version}</span></div>
                </div>
                <button className="admin-sidebar-collapse-button items-center justify-center whitespace-nowrap text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 border border-input bg-background shadow-sm hover:bg-accent hover:text-accent-foreground h-9 w-9 absolute -right-5 top-1/2 hidden rounded-full md:inline-flex" aria-label={tCommon("toggleSidebar")} onClick={onToggleCollapsed}>
                    <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" className={classNames("h-5 w-5", collapsed && "rotate-180")}><path d="M11 7l-5 5l5 5" /><path d="M17 7l-5 5l5 5" /></svg>
                </button>
            </div>
        </aside>
    );
}
