import React, { useEffect, useState } from "react";
import { Sidebar } from "../../components/Sidebar.jsx";
import { useDocumentTitle } from "../../composables/useDocumentTitle.js";
import { useHashLocation } from "../../composables/useHashLocation.js";
import { DashboardView } from "./DashboardView.jsx";
import { SettingsView } from "../config/SettingsView.jsx";
import { PluginManagement } from "../config/PluginManagement.jsx";
import { ThemeConfig } from "../config/ThemeConfig.jsx";
import { NoticeManagement } from "../config/NoticeManagement.jsx";
import { PaymentConfig } from "../config/PaymentConfig.jsx";
import { KnowledgeManagement } from "../config/KnowledgeManagement.jsx";
import { GroupManagement } from "../config/GroupManagement.jsx";
import { RouteManagement } from "../config/RouteManagement.jsx";
import { CouponManagement } from "../config/CouponManagement.jsx";
import { PlanManagement } from "../config/PlanManagement.jsx";
import { MachineManagement } from "../config/MachineManagement.jsx";
import { NodeManagement } from "../config/NodeManagement.jsx";
import { OrderManagement } from "../config/OrderManagement.jsx";
import { GiftCardManagement } from "../config/GiftCardManagement.jsx";
import { UserManagement } from "../config/UserManagement.jsx";
import { TicketManagement } from "../config/TicketManagement.jsx";

const SIDEBAR_COLLAPSED_KEY = "collapsed-sidebar";

function readSidebarCollapsed() {
    if (typeof window === "undefined") {
        return false;
    }

    try {
        return window.localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === "true";
    } catch {
        return false;
    }
}

function writeSidebarCollapsed(value) {
    try {
        window.localStorage.setItem(SIDEBAR_COLLAPSED_KEY, String(value));
    } catch {
        // Ignore storage failures; sidebar still works for current session.
    }
}

export function HomeView() {
    const title = globalThis.window?.settings?.title || "XBoard";
    const location = useHashLocation();
    const [sidebarCollapsed, setSidebarCollapsed] = useState(readSidebarCollapsed);
    const [mobileSidebarOpen, setMobileSidebarOpen] = useState(false);

    useDocumentTitle(title);

    useEffect(() => {
        document.body.classList.toggle("admin-mobile-sidebar-open", mobileSidebarOpen);
        return () => document.body.classList.remove("admin-mobile-sidebar-open");
    }, [mobileSidebarOpen]);

    return (
        <div className="app-shell min-h-svh bg-background text-foreground" data-sidebar-collapsed={sidebarCollapsed} data-mobile-sidebar-open={mobileSidebarOpen}>
            <Sidebar collapsed={sidebarCollapsed} onToggleCollapsed={() => setSidebarCollapsed((value) => {
                const nextValue = !value;
                writeSidebarCollapsed(nextValue);
                return nextValue;
            })} onMobileOpenChange={setMobileSidebarOpen} />
            <main className="admin-shell-main min-h-svh">
                {location.pathname === "/" ? <DashboardView /> : null}
                {location.pathname.startsWith("/config/system") ? <SettingsView /> : null}
                {location.pathname === "/config/plugin" ? <PluginManagement /> : null}
                {location.pathname === "/config/theme" ? <ThemeConfig /> : null}
                {location.pathname === "/config/notice" ? <NoticeManagement /> : null}
                {location.pathname === "/config/payment" ? <PaymentConfig /> : null}
                {location.pathname === "/config/knowledge" ? <KnowledgeManagement /> : null}
                {location.pathname === "/server/group" ? <GroupManagement /> : null}
                {location.pathname === "/server/route" ? <RouteManagement /> : null}
                {location.pathname === "/finance/coupon" ? <CouponManagement /> : null}
                {location.pathname === "/finance/plan" ? <PlanManagement /> : null}
                {location.pathname === "/server/machine" ? <MachineManagement /> : null}
                {location.pathname === "/server/manage" ? <NodeManagement /> : null}
                {location.pathname === "/finance/order" ? <OrderManagement /> : null}
                {location.pathname === "/finance/gift-card" ? <GiftCardManagement /> : null}
                {location.pathname === "/user/manage" ? <UserManagement /> : null}
                {location.pathname === "/user/ticket" ? <TicketManagement /> : null}
            </main>
        </div>
    );
}
