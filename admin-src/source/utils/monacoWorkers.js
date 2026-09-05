const WORKER_BY_LABEL = {
    editorWorkerService: "editor.worker.js",
    "monaco-editor-worker": "editor.worker.js",
    css: "css.worker.js",
    less: "css.worker.js",
    scss: "css.worker.js",
    html: "html.worker.js",
    handlebars: "html.worker.js",
    razor: "html.worker.js",
    json: "json.worker.js",
    javascript: "ts.worker.js",
    typescript: "ts.worker.js",
};

function getModuleDirectoryUrl() {
    const moduleUrl = import.meta.url;
    return moduleUrl.slice(0, moduleUrl.lastIndexOf("/") + 1);
}

function getWorkerBaseUrl() {
    if (import.meta.env.DEV) {
        return `${window.location.origin}${import.meta.env.BASE_URL}assets/workers/`;
    }

    return `${getModuleDirectoryUrl()}workers/`;
}

export function resolveMonacoWorkerFilename(label = "") {
    return WORKER_BY_LABEL[label] || null;
}

export function getMonacoWorkerUrl(label = "") {
    const filename = resolveMonacoWorkerFilename(label);
    if (!filename) {
        throw new Error(`No bundled Monaco worker for label: ${label}`);
    }

    return `${getWorkerBaseUrl()}${filename}`;
}

export function setupMonacoWorkers() {
    if (typeof globalThis === "undefined") {
        return;
    }

    const previousEnvironment = globalThis.MonacoEnvironment || {};

    globalThis.MonacoEnvironment = {
        ...previousEnvironment,
        getWorkerUrl(moduleId, label) {
            if (typeof previousEnvironment.getWorkerUrl === "function") {
                return previousEnvironment.getWorkerUrl(moduleId, label);
            }

            return getMonacoWorkerUrl(label);
        },
        getWorker(moduleId, label) {
            if (typeof previousEnvironment.getWorker === "function") {
                return previousEnvironment.getWorker(moduleId, label);
            }

            return new Worker(getMonacoWorkerUrl(label), {
                name: label,
                type: "module",
            });
        },
    };
}
