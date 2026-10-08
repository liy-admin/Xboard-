import { http, getSecurePath } from "./http.js";

const sp = () => getSecurePath();

// Node management
export function getNodes() {
    return http.get(sp() + "/server/manage/getNodes");
}

// 节点变更后的客户端覆盖情况：有多少人已经同步到最新节点集
export function getNodeSyncStats() {
    return http.get(sp() + "/server/manage/syncStats");
}

// 手动触发测速（后台异步）：传 ids 测这几个；不传测全部
export function probeNodeLatency(ids) {
    return http.post(sp() + "/server/manage/probeLatency", ids && ids.length ? { ids } : {});
}

// 测速进度（供进度条轮询，刷新后仍可见）
export function getNodeProbeStatus() {
    return http.get(sp() + "/server/manage/probeStatus");
}

export function saveNode(data) {
    return http.post(sp() + "/server/manage/save", data);
}

export function dropNode(data) {
    return http.post(sp() + "/server/manage/drop", data);
}

export function batchDeleteNodes(data) {
    return http.post(sp() + "/server/manage/batchDelete", data);
}

export function batchUpdateNodes(data) {
    return http.post(sp() + "/server/manage/batchUpdate", data);
}

export function copyNode(data) {
    return http.post(sp() + "/server/manage/copy", data);
}

export function updateNode(data) {
    return http.post(sp() + "/server/manage/update", data);
}

export function sortNodes(data) {
    return http.post(sp() + "/server/manage/sort", data);
}

export function resetNodeTraffic(data) {
    return http.post(sp() + "/server/manage/resetTraffic", data);
}

export function batchResetNodeTraffic(data) {
    return http.post(sp() + "/server/manage/batchResetTraffic", data);
}

export function generateEchKey(publicName) {
    return http.get(sp() + "/server/manage/generateEchKey", { params: { public_name: publicName } });
}

export function parseNodes(data) {
    return http.post(sp() + "/server/manage/parseNodes", data);
}

export function batchImportNodes(data) {
    return http.post(sp() + "/server/manage/batchImport", data);
}

// Permission groups
export function fetchGroups() {
    return http.get(sp() + "/server/group/fetch");
}

export function saveGroup(data) {
    return http.post(sp() + "/server/group/save", data);
}

export function dropGroup(data) {
    return http.post(sp() + "/server/group/drop", data);
}

// Routes
export function fetchRoutes() {
    return http.get(sp() + "/server/route/fetch");
}

export function saveRoute(data) {
    return http.post(sp() + "/server/route/save", data);
}

export function dropRoute(data) {
    return http.post(sp() + "/server/route/drop", data);
}

// Machines
export function fetchMachines() {
    return http.get(sp() + "/server/machine/fetch");
}

export function saveMachine(data) {
    return http.post(sp() + "/server/machine/save", data);
}

export function getMachineToken(params) {
    return http.get(sp() + "/server/machine/getToken", { params });
}

export function getMachineInstallCommand(params) {
    return http.get(sp() + "/server/machine/installCommand", { params });
}

export function resetMachineToken(data) {
    return http.post(sp() + "/server/machine/resetToken", data);
}

export function dropMachine(data) {
    return http.post(sp() + "/server/machine/drop", data);
}

export function getMachineNodes(params) {
    return http.get(sp() + "/server/machine/nodes", { params });
}

export function getMachineHistory(id, duration = 360, params) {
    return http.get(sp() + "/server/machine/history", { params: { id, duration, ...params } });
}
