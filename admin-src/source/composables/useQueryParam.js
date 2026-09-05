import { useMemo } from "react";
import { useHashLocation } from "./useHashLocation.js";
import { readQueryParam } from "../utils/query.js";

export function useQueryParam(name, fallback = "") {
    const location = useHashLocation();

    return useMemo(() => {
        return readQueryParam(name, location) || fallback;
    }, [fallback, location.hash, location.search, name]);
}
