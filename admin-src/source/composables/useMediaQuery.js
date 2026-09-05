import { useEffect, useState } from "react";

export function useMediaQuery(query) {
    const [matches, setMatches] = useState(() => {
        if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
            return false;
        }

        return window.matchMedia(query).matches;
    });

    useEffect(() => {
        if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
            return undefined;
        }

        const mediaQueryList = window.matchMedia(query);
        const updateMatches = (event) => setMatches(event.matches);

        setMatches(mediaQueryList.matches);

        if (typeof mediaQueryList.addEventListener === "function") {
            mediaQueryList.addEventListener("change", updateMatches);
            return () => mediaQueryList.removeEventListener("change", updateMatches);
        }

        mediaQueryList.addListener(updateMatches);
        return () => mediaQueryList.removeListener(updateMatches);
    }, [query]);

    return matches;
}
