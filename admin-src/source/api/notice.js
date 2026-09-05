import { http, getSecurePath } from "./http.js";

const sp = () => getSecurePath();

export function fetchNotices() {
    return http.get(`${sp()}/notice/fetch`);
}

export function saveNotice(data) {
    return http.post(`${sp()}/notice/save`, data);
}

export function dropNotice(id) {
    return http.post(`${sp()}/notice/drop`, { id });
}

export function showNotice(id) {
    return http.post(`${sp()}/notice/show`, { id });
}

export function sortNotices(ids) {
    return http.post(`${sp()}/notice/sort`, { ids });
}
