import { http, getSecurePath } from "./http.js";

const sp = () => getSecurePath();

export function fetchThemes() {
    return http.get(sp() + "/theme/getThemes");
}

export function getThemeConfig(name) {
    return http.post(sp() + "/theme/getThemeConfig", { name });
}

export function saveThemeConfig(name, config) {
    return http.post(sp() + "/theme/saveThemeConfig", { name, config });
}

export function uploadTheme(file) {
    const form = new FormData();
    form.append("file", file);
    return http.post(sp() + "/theme/upload", form, { headers: { "Content-Type": "multipart/form-data" } });
}

export function deleteTheme(name) {
    return http.post(sp() + "/theme/delete", { name });
}
