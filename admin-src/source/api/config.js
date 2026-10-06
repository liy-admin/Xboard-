import { http, getSecurePath } from "./http.js";

const sp = () => getSecurePath();

export function getSettings(key = "") {
    return http.get(sp() + "/config/fetch?key=" + key);
}

export function saveSettings(settings) {
    return http.post(sp() + "/config/save", settings);
}

export function uploadDiscoverHtml(file) {
    const form = new FormData();
    form.append("file", file);
    return http.post(sp() + "/config/uploadDiscoverHtml", form, { headers: { "Content-Type": "multipart/form-data" } });
}

export function getEmailTemplate() {
    return http.get(sp() + "/config/getEmailTemplate");
}

export function sendTestMail(email) {
    return http.post(sp() + "/config/testSendMail", email ? { email } : {});
}

export function setTelegramWebhook() {
    return http.post(sp() + "/config/setTelegramWebhook");
}

export function getSystemStatus() {
    return http.get(`${sp()}/system/getSystemStatus`);
}

export function getQueueStats() {
    return http.get(`${sp()}/system/getQueueStats`);
}

export function getQueueWorkload() {
    return http.get(`${sp()}/system/getQueueWorkload`);
}

export function getQueueMasters() {
    return http.get(`${sp()}/system/getQueueMasters`);
}

export function getHorizonFailedJobs(params) {
    return http.get(`${sp()}/system/getHorizonFailedJobs`, { params });
}

export function getReferralSummary() {
    return http.get(`${sp()}/referral/summary`);
}

export function getReferralRecords(params) {
    return http.get(`${sp()}/referral/records`, { params });
}
