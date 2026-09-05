import React, { useEffect } from "react";
import { useAuthGuard } from "./composables/useAuthGuard.js";
import { useHashLocation } from "./composables/useHashLocation.js";
import { isPublicPath } from "./router/index.js";
import { useAuthState } from "./store/auth.js";
import { loadSiteConfig } from "./store/config.js";
import { HomeView } from "./views/home/HomeView.jsx";
import { SignInView } from "./views/auth/SignInView.jsx";

const AUTH_PATHS = new Set([
    "/sign-in",
    "/sign-in-2",
    "/sign-up",
    "/forgot-password",
    "/otp",
]);

export function App() {
    const location = useHashLocation();
    const auth = useAuthState();

    useAuthGuard();

    useEffect(() => {
        if (auth.token) {
            loadSiteConfig();
        }
    }, [auth.token]);

    if (!auth.token && !isPublicPath(location.pathname)) {
        return null;
    }

    if (AUTH_PATHS.has(location.pathname)) {
        return <SignInView />;
    }

    return <HomeView />;
}
