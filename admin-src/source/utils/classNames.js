export function classNames(...parts) {
    const flattened = parts.flat(Infinity);
    const tokens = [];

    for (const part of flattened) {
        if (!part) {
            continue;
        }

        if (typeof part === "string") {
            tokens.push(part);
            continue;
        }

        if (Array.isArray(part)) {
            tokens.push(classNames(...part));
            continue;
        }

        if (typeof part === "object") {
            for (const [key, value] of Object.entries(part)) {
                if (value) {
                    tokens.push(key);
                }
            }
        }
    }

    return tokens.filter(Boolean).join(" ");
}
