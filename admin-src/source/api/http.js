import axios from "axios";
import { getAuthState } from "../store/auth.js";
import { redirectToSignIn } from "../router/index.js";

function getBaseUrl() {
    const base = globalThis.window?.settings?.base_url || "/";
    return base.endsWith("/") ? `${base}api/v2` : `${base}/api/v2`;
}

const PUBLIC_PATHS = [
    "/passport/auth/login",
    "/passport/auth/token2Login",
    "/passport/auth/register",
    "/guest/comm/config",
    "/passport/comm/sendEmailVerify",
    "/passport/auth/forget",
];

export const http = axios.create({
    baseURL: getBaseUrl(),
    timeout: 30000,
    headers: { "Content-Type": "application/json" },
});

http.interceptors.request.use(
    (config) => {
        const urlPath = config.url?.split("?")[0] || "";
        if (!PUBLIC_PATHS.includes(urlPath)) {
            const { token } = getAuthState();
            if (!token) {
                redirectToSignIn();
                return Promise.reject({ code: -1, message: "Not logged in" });
            }
            config.headers.Authorization = token;
        }
        config.headers["Content-Language"] =
            (typeof localStorage !== "undefined" && localStorage.getItem("i18nextLng")) || "zh-CN";
        return config;
    },
    (error) => Promise.reject(error),
);

http.interceptors.response.use(
    (response) => response?.data || { code: -1, message: "Unknown error" },
    (error) => {
        const status = error.response?.status;
        const message = error.response?.data?.message;
        if (status === 401 || status === 403) {
            redirectToSignIn();
        }
        return Promise.reject(
            error.response?.data || { data: null, code: -1, message: message || error.message || "Unknown error" },
        );
    },
);

export function getSecurePath() {
    return globalThis.window?.settings?.secure_path ?? "";
}

export class HttpError extends Error {
    constructor(message, status = 500, payload = null) {
        super(message);
        this.name = "HttpError";
        this.status = status;
        this.payload = payload;
    }
}
