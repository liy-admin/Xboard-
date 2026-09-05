import { http, getSecurePath } from "./http.js";

const sp = () => getSecurePath();

export function fetchPayments() {
    return http.get(sp() + "/payment/fetch");
}

export function getPaymentMethods() {
    return http.get(sp() + "/payment/getPaymentMethods");
}

export function getPaymentForm(data) {
    return http.post(sp() + "/payment/getPaymentForm", data);
}

export function savePayment(data) {
    return http.post(sp() + "/payment/save", data);
}

export function dropPayment(data) {
    return http.post(sp() + "/payment/drop", data);
}

export function showPayment(data) {
    return http.post(sp() + "/payment/show", data);
}

export function sortPayments(data) {
    return http.post(sp() + "/payment/sort", data);
}

export function uploadPaymentIcon(file) {
    const form = new FormData();
    form.append("file", file);
    return http.post(sp() + "/payment/uploadIcon", form, { headers: { "Content-Type": "multipart/form-data" } });
}
