import React from "react";
import { SettingsSidebarContent } from "./SettingsLayout.jsx";
import { useHashLocation } from "../../composables/useHashLocation.js";
import { useTranslation } from "../../i18n/index.js";
import { useAuthState } from "../../store/auth.js";
import { HeaderControls } from "../home/DashboardView.jsx";
import { SiteSettings } from "./SiteSettings.jsx";
import { WebsiteSettings } from "./WebsiteSettings.jsx";
import { SafeSettings } from "./SafeSettings.jsx";
import { SubscribeSettings } from "./SubscribeSettings.jsx";
import { InviteSettings } from "./InviteSettings.jsx";
import { LuckyDrawSettings } from "./LuckyDrawSettings.jsx";
import { DiscoverSettings } from "./DiscoverSettings.jsx";
import { AntiLossSettings } from "./AntiLossSettings.jsx";
import { AdBannerSettings } from "./AdBannerSettings.jsx";
import { SupportSettings } from "./SupportSettings.jsx";
import { FrontendSettings } from "./FrontendSettings.jsx";
import { ServerSettings } from "./ServerSettings.jsx";
import { EmailSettings } from "./EmailSettings.jsx";
import { TelegramSettings } from "./TelegramSettings.jsx";
import { AppSettings } from "./AppSettings.jsx";
import { SubscribeTemplateSettings } from "./SubscribeTemplateSettings.jsx";

const TAB_MAP = {
    "/config/system": SiteSettings,
    "/config/system/website": WebsiteSettings,
    "/config/system/safe": SafeSettings,
    "/config/system/subscribe": SubscribeSettings,
    "/config/system/invite": InviteSettings,
    "/config/system/lucky-draw": LuckyDrawSettings,
    "/config/system/discover": DiscoverSettings,
    "/config/system/anti-loss": AntiLossSettings,
    "/config/system/ad-banner": AdBannerSettings,
    "/config/system/support": SupportSettings,
    "/config/system/frontend": FrontendSettings,
    "/config/system/server": ServerSettings,
    "/config/system/email": EmailSettings,
    "/config/system/telegram": TelegramSettings,
    "/config/system/app": AppSettings,
    "/config/system/subscribe-template": SubscribeTemplateSettings,
};

export function SettingsView() {
    const location = useHashLocation();
    const auth = useAuthState();
    const { t } = useTranslation("settings");
    const TabContent = TAB_MAP[location.pathname] || SiteSettings;

    return (
        <div className="relative flex h-full w-full flex-col">
            <div className="flex h-[var(--header-height)] flex-none items-center gap-4 bg-background p-4 md:px-8">
                <HeaderControls auth={auth} />
            </div>
            <div className="flex-1 overflow-auto px-4 pb-10 md:px-8">
                <div className="space-y-6">
                    <div className="space-y-2 pt-8">
                        <h1 className="text-4xl font-bold tracking-tight">{t("title")}</h1>
                        <p className="text-lg text-muted-foreground">{t("description")}</p>
                    </div>
                    <hr />
                    <div className="flex flex-col gap-8 lg:flex-row lg:gap-12">
                        <aside className="lg:w-1/5 lg:shrink-0">
                            <SettingsSidebarContent currentPath={location.pathname} />
                        </aside>
                        <div className="min-w-0 flex-1 pb-16">
                            <TabContent />
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
}
