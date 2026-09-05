import { http, getSecurePath } from "./http.js";

const sp = () => getSecurePath();

export function fetchUsers(params) {
    return http.post(`${sp()}/user/fetch`, params);
}

export function updateUser(data) {
    return http.post(`${sp()}/user/update`, data);
}

export function resetUserSecret(id) {
    return http.post(`${sp()}/user/resetSecret`, { id });
}

export function generateUsers(params) {
    if (params.download_csv) {
        return http.post(`${sp()}/user/generate`, params, { responseType: "blob" });
    }
    return http.post(`${sp()}/user/generate`, params);
}

export function getStatUser(params) {
    return http.post(`${sp()}/stat/getStatUser`, params);
}

export function destroyUser(id) {
    return http.post(`${sp()}/user/destroy`, { id });
}

export function sendUserMail(data) {
    return http.post(`${sp()}/user/sendMail`, data);
}

export function dumpUserCSV(params) {
    return http.post(`${sp()}/user/dumpCSV`, params, { responseType: "blob" });
}

export function banUser(data) {
    return http.post(`${sp()}/user/ban`, data);
}

export function fetchTrafficResetLogs(params) {
    return http.get(`${sp()}/traffic-reset/logs`, { params });
}

export function resetUserTraffic(data) {
    return http.post(`${sp()}/traffic-reset/reset-user`, data);
}

export function fetchUserTrafficHistory(userId, params) {
    return http.get(`${sp()}/traffic-reset/user/${userId}/history`, { params });
}

export function getOnlineUsers() {
    return http.get(`${sp()}/user/getOnlineUsers`);
}
