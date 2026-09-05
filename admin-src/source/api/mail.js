import { http, getSecurePath } from "./http.js";

const sp = () => getSecurePath();

export const mailTemplate = {
    list: () => http.get(`${sp()}/mail/template/list`),
    get: (name) => http.get(`${sp()}/mail/template/get?name=${name}`),
    save: (data) => http.post(`${sp()}/mail/template/save`, data),
    reset: (name) => http.post(`${sp()}/mail/template/reset`, { name }),
    test: (name, email) => http.post(`${sp()}/mail/template/test`, { name, email }),
};
