function createMemoryStorage() {
    const items = new Map();

    return {
        get length() {
            return items.size;
        },
        key(index) {
            return Array.from(items.keys())[index] ?? null;
        },
        getItem(key) {
            return items.has(key) ? items.get(key) : null;
        },
        setItem(key, value) {
            items.set(key, String(value));
        },
        removeItem(key) {
            items.delete(key);
        },
        clear() {
            items.clear();
        },
    };
}

function resolveNativeStorage(storageName) {
    if (typeof globalThis === "undefined") {
        return null;
    }

    const storage = globalThis[storageName];
    if (!storage || typeof storage.getItem !== "function") {
        return null;
    }

    return storage;
}

export function createPrefixedStorage({ prefix = "", storage = resolveNativeStorage("localStorage") } = {}) {
    const backend = storage ?? createMemoryStorage();

    const makeKey = (key) => `${prefix}${key}`;

    return {
        get(key, fallback = null) {
            const value = backend.getItem(makeKey(key));
            return value === null || value === undefined ? fallback : value;
        },
        set(key, value) {
            backend.setItem(makeKey(key), String(value));
            return String(value);
        },
        remove(key) {
            backend.removeItem(makeKey(key));
        },
        clear() {
            if (typeof backend.length === "number" && typeof backend.key === "function") {
                const keys = [];
                for (let index = 0; index < backend.length; index += 1) {
                    const key = backend.key(index);
                    if (key && key.startsWith(prefix)) {
                        keys.push(key);
                    }
                }

                keys.forEach((key) => backend.removeItem(key));
                return;
            }

            backend.clear();
        },
        raw: backend,
    };
}

export const authStorage = createPrefixedStorage({ prefix: "Xboard_" });
