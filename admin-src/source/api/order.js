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
