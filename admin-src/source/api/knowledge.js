import { http, getSecurePath } from "./http.js";

const sp = () => getSecurePath();

export function fetchKnowledge() {
    return http.get(sp() + "/knowledge/fetch");
}

export function fetchKnowledgeById(id) {
    return http.get(sp() + "/knowledge/fetch?id=" + id);
}

export function saveKnowledge(data) {
    return http.post(sp() + "/knowledge/save", data);
}

export function dropKnowledge(data) {
    return http.post(sp() + "/knowledge/drop", data);
}

export function showKnowledge(data) {
    return http.post(sp() + "/knowledge/show", data);
}

export function sortKnowledge(data) {
    return http.post(sp() + "/knowledge/sort", data);
}
