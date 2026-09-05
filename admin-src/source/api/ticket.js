import { http, getSecurePath } from "./http.js";

const sp = () => getSecurePath();

export function fetchTickets(params) {
    return http.post(sp() + "/ticket/fetch", params);
}

export function fetchTicketById(id) {
    return http.get(sp() + "/ticket/fetch?id=" + id);
}

export function replyTicket(data) {
    return http.post(sp() + "/ticket/reply", data);
}

export function closeTicket(id) {
    return http.post(sp() + "/ticket/close", { id });
}
