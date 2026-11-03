import { http, getSecurePath } from "./http.js";

const sp = () => getSecurePath();

export function fetchOrders(params) {
    return http.post(sp() + "/order/fetch", params);
}

export function getOrderDetail(data) {
    return http.post(sp() + "/order/detail", data);
}

export function payOrder(data) {
    return http.post(sp() + "/order/paid", data);
}

export function cancelOrder(data) {
    return http.post(sp() + "/order/cancel", data);
}

export function updateOrder(data) {
    return http.post(sp() + "/order/update", data);
}

export function assignOrder(data) {
    return http.post(sp() + "/order/assign", data);
}

// 已取消订单自动清理：状态（设置 / 待清理数 / 上次结果）、保存设置、立即清理
export function getCleanupStatus() {
    return http.get(sp() + "/order/cleanupStatus");
}

export function saveCleanupSettings(data) {
    return http.post(sp() + "/order/cleanupSave", data);
}

export function runCleanup(data) {
    return http.post(sp() + "/order/cleanupRun", data);
}
