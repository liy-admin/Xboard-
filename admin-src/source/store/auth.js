import { useSyncExternalStore } from "react";
import { authStorage } from "../utils/storage.js";

const ACCESS_TOKEN_KEY = "access_token";

const listeners = new Set();

let currentState = {
    token: authStorage.get(ACCESS_TOKEN_KEY, ""),
    userInfo: null,
    isLoggedIn: Boolean(authStorage.get(ACCESS_TOKEN_KEY, "")),
    loading: false,
    error: null,
};

function emit() {
    currentState = { ...currentState };
    listeners.forEach((listener) => listener());
}

export function getAuthState() {
    return currentState;
}

export function subscribeAuth(listener) {
    listeners.add(listener);
    return () => {
        listeners.delete(listener);
    };
}

export function useAuthState() {
    return useSyncExternalStore(
        subscribeAuth,
        getAuthState,
        getAuthState,
    );
}

export function setAuthToken(token) {
    const value = token ? String(token) : "";
    currentState.token = value;
    currentState.isLoggedIn = Boolean(value);

    if (value) {
        authStorage.set(ACCESS_TOKEN_KEY, value);
    } else {
        authStorage.remove(ACCESS_TOKEN_KEY);
    }

    emit();
    return value;
}

export function setUserInfo(userInfo) {
    currentState.userInfo = userInfo ?? null;
    emit();
    return currentState.userInfo;
}

export function setAuthLoading(loading) {
    currentState.loading = Boolean(loading);
    emit();
}

export function setAuthError(error) {
    currentState.error = error || null;
    emit();
}

export function clearAuthSession() {
    authStorage.remove(ACCESS_TOKEN_KEY);
    currentState.token = "";
    currentState.userInfo = null;
    currentState.isLoggedIn = false;
    currentState.loading = false;
    currentState.error = null;
    emit();
}
