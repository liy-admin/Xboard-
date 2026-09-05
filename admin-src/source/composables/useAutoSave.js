import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import isEqual from "lodash/isEqual";

export function useAutoSave({ debounceMs = 1000, onSave, loaded = true }) {
    const [saving, setSaving] = useState(false);
    const lastSavedRef = useRef(null);

    const debouncedSave = useMemo(() => {
        let timer = null;
        let cancelled = false;

        const fn = async (data) => {
            if (!isEqual(data, lastSavedRef.current)) {
                setSaving(true);
                try {
                    await onSave(data);
                    lastSavedRef.current = { ...data };
                } finally {
                    if (!cancelled) {
                        setSaving(false);
                    }
                }
            }
        };

        fn.cancel = () => {
            cancelled = true;
            if (timer) {
                clearTimeout(timer);
            }
        };

        return fn;
    }, [onSave, debounceMs]);

    const save = useCallback(
        (data) => {
            if (!loaded) return;
            debouncedSave(data);
        },
        [debouncedSave, loaded],
    );

    const markLoaded = useCallback((data) => {
        lastSavedRef.current = { ...data };
    }, []);

    useEffect(() => {
        return () => debouncedSave.cancel();
    }, [debouncedSave]);

    return { saving, save, markLoaded };
}
