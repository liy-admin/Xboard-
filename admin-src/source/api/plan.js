import { http, getSecurePath } from "./http.js";

const sp = () => getSecurePath();

export function fetchPlans() {
    return http.get(sp() + "/plan/fetch");
}

export function savePlan(data) {
    return http.post(sp() + "/plan/save", data);
}

export function updatePlan(data) {
    return http.post(sp() + "/plan/update", data);
}

export function dropPlan(data) {
    return http.post(sp() + "/plan/drop", data);
}

export function sortPlans(ids) {
    return http.post(sp() + "/plan/sort", { ids });
}
