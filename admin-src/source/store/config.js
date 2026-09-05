import { useSyncExternalStore } from "react";
import { getSettings } from "../api/config.js";

const listeners = new Set();

let currentState = {
    currencySymbol: "¥",
    loaded: false,
};

function emit() {
    currentState = { ...currentState };
    listeners.forEach((listener) => listener());
}

export function getConfigState() {
    return currentState;
}

export function getCurrencySymbol() {
    return currentState.currencySymbol;
}

export function subscribeConfig(listener) {
    listeners.add(listener);
    return () => {
        listeners.delete(listener);
    };
}

export function useCurrencySymbol() {
    return useSyncExternalStore(subscribeConfig, getConfigState, getConfigState).currencySymbol;
}

let loadPromise = null;

export function loadSiteConfig() {
    if (loadPromise) return loadPromise;
    loadPromise = getSettings("site")
        .then((res) => {
            const site = res?.data?.site || res?.data || {};
            currentState.currencySymbol = site.currency_symbol || currentState.currencySymbol;
            currentState.loaded = true;
            emit();
        })
        .catch(() => {
            loadPromise = null;
        });
    return loadPromise;
}
