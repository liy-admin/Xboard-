const THEME_STORAGE_KEY = "theme";
const LEGACY_THEME_STORAGE_KEY = "vite-ui-theme";

function getStorageValue(key) {
    try {
        return window.localStorage.getItem(key);
    } catch {
        return null;
    }
}

function setStorageValue(key, value) {
    try {
        window.localStorage.setItem(key, value);
    } catch {
        // Ignore storage errors so the UI can still switch theme in-memory.
    }
}

function prefersDarkTheme() {
    return typeof window !== "undefined" && window.matchMedia?.("(prefers-color-scheme: dark)").matches;
}

export function getStoredTheme() {
    if (typeof window === "undefined") {
        return "light";
    }

    const storedTheme = getStorageValue(THEME_STORAGE_KEY) || getStorageValue(LEGACY_THEME_STORAGE_KEY);
    return ["dark", "light", "system"].includes(storedTheme) ? storedTheme : "light";
}

export function resolveTheme(theme = getStoredTheme()) {
    if (theme === "system") {
        return prefersDarkTheme() ? "dark" : "light";
    }

    return theme === "dark" ? "dark" : "light";
}

export function applyTheme(theme = getStoredTheme()) {
    const resolvedTheme = resolveTheme(theme);

    if (typeof document !== "undefined") {
        document.documentElement.classList.remove("light", "dark");
        document.documentElement.classList.add(resolvedTheme);
    }

    return resolvedTheme;
}

export function persistTheme(theme) {
    const nextTheme = theme === "dark" ? "dark" : "light";
    setStorageValue(THEME_STORAGE_KEY, nextTheme);
    setStorageValue(LEGACY_THEME_STORAGE_KEY, nextTheme);
    return applyTheme(nextTheme);
}
