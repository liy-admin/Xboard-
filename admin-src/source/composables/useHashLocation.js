import { useSyncExternalStore } from "react";
import { getCurrentLocation, subscribeLocation } from "../router/index.js";

export function useHashLocation() {
    return useSyncExternalStore(subscribeLocation, getCurrentLocation, getCurrentLocation);
}
