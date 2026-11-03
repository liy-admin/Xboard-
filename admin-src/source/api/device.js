import { http, getSecurePath } from "./http.js";

const sp = () => getSecurePath();

export function fetchDevices(params) {
    return http.post(sp() + "/device/fetch", params);
}

export function fetchDeviceDetail(deviceId) {
    return http.post(sp() + "/device/detail", { device_id: deviceId });
}

export function setDeviceTrialClaimed(deviceId, claimed) {
    return http.post(sp() + "/device/setTrialClaimed", { device_id: deviceId, claimed_trial: claimed ? 1 : 0 });
}
