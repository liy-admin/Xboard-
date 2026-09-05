import { http, getSecurePath } from "./http.js";

const sp = () => getSecurePath();

export function fetchGiftCardTemplates(params) {
    return http.post(sp() + "/gift-card/templates", params);
}

export function createGiftCardTemplate(data) {
    return http.post(sp() + "/gift-card/create-template", data);
}

export function updateGiftCardTemplate(data) {
    return http.post(sp() + "/gift-card/update-template", data);
}

export function deleteGiftCardTemplate(data) {
    return http.post(sp() + "/gift-card/delete-template", data);
}

export function fetchGiftCardCodes(params) {
    return http.post(sp() + "/gift-card/codes", params);
}

export function generateGiftCardCodes(params) {
    if (params.download_csv) {
        return http.post(sp() + "/gift-card/generate-codes", params, { responseType: "blob" });
    }
    return http.post(sp() + "/gift-card/generate-codes", params);
}

export function toggleGiftCardCode(data) {
    return http.post(sp() + "/gift-card/toggle-code", data);
}

export function fetchGiftCardUsages(params) {
    return http.post(sp() + "/gift-card/usages", params);
}

export function fetchGiftCardStatistics(params) {
    return http.post(sp() + "/gift-card/statistics", params);
}
