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

// 订阅导入预设
export function getImportSubscriptions() {
    return http.get(sp() + "/server/manage/importSubscriptions");
}

export function saveImportSubscriptions(subscriptions) {
    return http.post(sp() + "/server/manage/importSubscriptions", { subscriptions });
}

// 带某个订阅标识的节点数（删除前确认用）
export function countNodesByTag(tag) {
    return http.get(sp() + "/server/manage/countByTag?tag=" + encodeURIComponent(tag));
}

// 删除带某个订阅标识的全部节点（订阅预设保留）
export function deleteNodesByTag(tag) {
    return http.post(sp() + "/server/manage/deleteByTag", { tag });
}

// 停止正在进行的测速（当前节点测完后停止）
export function stopNodeProbe() {
    return http.post(sp() + "/server/manage/probeStop", {});
}

// 从 Balala 订阅抓取插件获取最近一次订阅链接
export function getPluginSubscription() {
    return http.get(sp() + "/server/manage/pluginSubscription");
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

// 一键为服务器创建 VLESS + Reality 节点
export function quickCreateMachineNode(data) {
    return http.post(sp() + "/server/machine/quickNode", data);
}

// 实测服务器上的一个节点是否可用（同步，几秒到十几秒）
export function testMachineNode(id) {
    return http.post(sp() + "/server/machine/testNode", { id }, { timeout: 60000 });
}

// 服务器上全部节点的分享链接；email 为空时每个节点用其权限组里的第一个可用用户
export function getMachineNodeLinks(machineId, email) {
    return http.get(sp() + "/server/machine/nodeLinks", { params: { id: machineId, email: email || undefined } });
}
