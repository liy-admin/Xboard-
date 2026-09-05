import { http } from "./http.js";

export function signIn(credentials) {
    return http.post("/passport/auth/login", credentials);
}

export function register(data) {
    return http.post("/passport/auth/register", data);
}

export function forgotPassword(data) {
    return http.post("/passport/auth/forget", data);
}

export function sendEmailVerify(data) {
    return http.post("/passport/comm/sendEmailVerify", data);
}

export function token2Login(data) {
    return http.post("/passport/auth/token2Login", data);
}

export function fetchUserInfo() {
    return http.get("/user/info");
}

export function extractAuthToken(response) {
    return (
        response?.data?.auth_data ||
        response?.data?.token ||
        response?.auth_data ||
        response?.token ||
        ""
    );
}

export function extractUserInfo(response) {
    return response?.data ?? response ?? null;
}
