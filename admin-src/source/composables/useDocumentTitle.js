import { useEffect } from "react";

export function useDocumentTitle(title) {
    useEffect(() => {
        if (typeof document === "undefined") {
            return undefined;
        }

        const previousTitle = document.title;
        document.title = title || previousTitle;

        return () => {
            if (previousTitle) {
                document.title = previousTitle;
            }
        };
    }, [title]);
}
