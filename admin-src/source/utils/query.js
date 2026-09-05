export function normalizePath(pathname = "/") {
    if (!pathname) {
        return "/";
    }

    const nextPath = pathname.startsWith("/") ? pathname : `/${pathname}`;
    return nextPath.replace(/\/+/g, "/") || "/";
}

export function normalizeSearch(search = "") {
    if (!search) {
        return "";
    }

    return search.startsWith("?") ? search : `?${search}`;
}

export function parseHashLocation(hash = "") {
    const rawHash = hash || (typeof window !== "undefined" ? window.location.hash : "");
    const raw = rawHash.startsWith("#") ? rawHash.slice(1) : rawHash;

    if (!raw) {
        return {
            pathname: "/",
            search: "",
            hash: "#/",
        };
    }

    const [pathnamePart, searchPart = ""] = raw.split("?");
    const pathname = normalizePath(pathnamePart);
    const search = searchPart ? `?${searchPart}` : "";

    return {
        pathname,
        search,
        hash: `#${pathname}${search}`,
    };
}

export function readQueryParam(name, location = parseHashLocation()) {
    const search = location.search || "";
    const params = new URLSearchParams(search.startsWith("?") ? search.slice(1) : search);
    return params.get(name);
}

export function buildHashHref(pathname, search = "") {
    return `#${normalizePath(pathname)}${normalizeSearch(search)}`;
}

export function buildRedirectQuery(location = parseHashLocation()) {
    return `?redirect=${encodeURIComponent(`${location.pathname}${location.search}`)}`;
}
