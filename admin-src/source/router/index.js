import { buildHashHref, buildRedirectQuery, normalizePath, normalizeSearch, parseHashLocation, readQueryParam } from "../utils/query.js";

export const AUTH_PUBLIC_PATHS = [
    "/sign-in",
    "/sign-in-2",
    "/sign-up",
    "/forgot-password",
    "/otp",
];

const listeners = new Set();

function emitLocationChange() {
    listeners.forEach((listener) => listener());
}

if (typeof window !== "undefined") {
    window.addEventListener("hashchange", emitLocationChange);
    window.addEventListener("popstate", emitLocationChange);
}

export function subscribeLocation(listener) {
    listeners.add(listener);
    return () => {
        listeners.delete(listener);
    };
}

let cachedHash = null;
let cachedLocation = null;

export function getCurrentLocation() {
    if (typeof window === "undefined") {
        return {
            pathname: "/",
            search: "",
            hash: "#/",
        };
    }

    if (cachedHash !== window.location.hash || !cachedLocation) {
        cachedHash = window.location.hash;
        cachedLocation = parseHashLocation(cachedHash);
    }

    return cachedLocation;
}

export function navigate(pathname = "/", options = {}) {
    const nextPath = normalizePath(pathname);
    const nextSearch = normalizeSearch(options.search || "");
    const nextHash = buildHashHref(nextPath, nextSearch);

    if (typeof window === "undefined") {
        return nextHash;
    }

    if (options.replace) {
        const nextHref = `${window.location.pathname}${window.location.search}${nextHash}`;
        window.history.replaceState(window.history.state, "", nextHref);
        emitLocationChange();
        return nextHash;
    }

    window.location.hash = `${nextPath}${nextSearch}`;
    emitLocationChange();
    return nextHash;
}

export function buildSignInUrl(location = getCurrentLocation()) {
    return `/sign-in${buildRedirectQuery(location)}`;
}

export function redirectToSignIn(location = getCurrentLocation(), options = {}) {
    return navigate("/sign-in", {
        search: buildRedirectQuery(location),
        replace: options.replace ?? true,
    });
}

export function isPublicPath(pathname = "") {
    return AUTH_PUBLIC_PATHS.includes(normalizePath(pathname));
}

export function getRedirectTarget(location = getCurrentLocation(), fallback = "/") {
    return readQueryParam("redirect", location) || fallback;
}
