import { http, getSecurePath } from "./http.js";

const sp = () => getSecurePath();

export function fetchCoupons(params) {
    return http.post(sp() + "/coupon/fetch", params);
}

export function generateCoupon(data) {
    return http.post(sp() + "/coupon/generate", data);
}

export function dropCoupon(data) {
    return http.post(sp() + "/coupon/drop", data);
}

export function updateCoupon(data) {
    return http.post(sp() + "/coupon/update", data);
}
