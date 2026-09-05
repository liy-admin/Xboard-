import { useEffect } from "react";
import { useHashLocation } from "./useHashLocation.js";
import { useAuthState } from "../store/auth.js";
import { isPublicPath, redirectToSignIn } from "../router/index.js";

export function useAuthGuard() {
    const location = useHashLocation();
    const auth = useAuthState();

    useEffect(() => {
        if (typeof window === "undefined") {
            return;
        }

        if (!auth.token && !isPublicPath(location.pathname)) {
            redirectToSignIn(location, { replace: true });
        }
    }, [auth.token, location.pathname, location.search]);

    return auth;
}
