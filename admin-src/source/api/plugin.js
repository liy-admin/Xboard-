import { http, getSecurePath } from "./http.js";

const sp = () => getSecurePath();

export function getPluginTypes() {
    return http.get(`${sp()}/plugin/types`);
}

export function getPluginList(type) {
    return http.get(`${sp()}/plugin/getPlugins`, { params: type ? { type } : {} });
}

export function uploadPlugin(file) {
    const form = new FormData();
    form.append("file", file);
    return http.post(`${sp()}/plugin/upload`, form, { headers: { "Content-Type": "multipart/form-data" } });
}

export function deletePlugin(code) {
    return http.post(`${sp()}/plugin/delete`, { code });
}

export function installPlugin(code) {
    return http.post(`${sp()}/plugin/install`, { code });
}

export function uninstallPlugin(code) {
    return http.post(`${sp()}/plugin/uninstall`, { code });
}

export function enablePlugin(code) {
    return http.post(`${sp()}/plugin/enable`, { code });
}

export function disablePlugin(code) {
    return http.post(`${sp()}/plugin/disable`, { code });
}

export function getPluginConfig(code) {
    return http.get(`${sp()}/plugin/config`, { params: { code } });
}

export function updatePluginConfig(code, config) {
    return http.post(`${sp()}/plugin/config`, { code, config });
}

export function upgradePlugin(code) {
    return http.post(`${sp()}/plugin/upgrade`, { code });
}

export function crudList(path, params) {
    return http.get(`${sp()}${path}`, { params });
}

export function crudSave(path, data) {
    return http.post(`${sp()}${path}`, data);
}

export function crudDelete(path, data) {
    return http.post(`${sp()}${path}`, data);
}
