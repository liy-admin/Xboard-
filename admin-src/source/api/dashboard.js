import { http, getSecurePath } from "./http.js";

const sp = () => getSecurePath();

export function fetchDashboardStats() {
    return http.get(sp() + "/stat/getStats");
}

export function fetchOrderStats(params = {}) {
    return http.get(sp() + "/stat/getOrder", { params });
}

export function fetchTrafficRank(params = {}) {
    return http.get(sp() + "/stat/getTrafficRank", { params });
}

export function fetchQueueStats() {
    return http.get(`${sp()}/system/getQueueStats`);
}

export function fetchQueueWorkload() {
    return http.get(`${sp()}/system/getQueueWorkload`);
}

export function fetchQueueMasters() {
    return http.get(`${sp()}/system/getQueueMasters`);
}

export function fetchHorizonFailedJobs(params) {
    return http.get(`${sp()}/system/getHorizonFailedJobs`, { params });
}
