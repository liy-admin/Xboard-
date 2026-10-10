import { http, getSecurePath } from "./http.js";

const sp = () => getSecurePath();

export function getSettings(key = "") {
    return http.get(sp() + "/config/fetch?key=" + key);
}

export function saveSettings(settings) {
    return http.post(sp() + "/config/save", settings);
}

// 诊断 sing-box 测速是否可用（检查/下载 sing-box + 对一个样本节点实跑一次）。
// 首次可能要下载 sing-box，单独放宽超时到 120s，避免默认 30s 误判超时。
export function diagnoseProbe() {
    return http.get(sp() + "/server/manage/probeDiagnose", { timeout: 120000 });
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
