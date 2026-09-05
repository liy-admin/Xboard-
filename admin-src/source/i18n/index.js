import React, { createContext, useContext, useEffect, useMemo, useState } from "react";

export const DEFAULT_LANGUAGE = "vi-VN";

export const LANGUAGE_OPTIONS = [
    {
        code: "en-US",
        name: "English",
        shortName: "EN",
    },
    {
        code: "zh-CN",
        name: "中文",
        shortName: "CN",
    },
    {
        code: "vi-VN",
        name: "Tiếng Việt",
        shortName: "VN",
    },
];

const FALLBACK_LOCALES = {
    "en-US": {
        auth: {
            signIn: {
                title: "Sign In",
                description: "Enter your email and password to sign in",
                email: "Email",
                emailPlaceholder: "name@example.com",
                password: "Password",
                passwordPlaceholder: "Enter your password",
                forgotPassword: "Forgot Password?",
                submit: "Sign In",
                rememberMe: "Remember me",
                resetPassword: {
                    title: "Reset Password",
                    description: "Execute the following command in the site directory to reset your password",
                    command: "php artisan reset:password admin-email",
                },
                validation: {
                    emailRequired: "Please enter your email address",
                    emailInvalid: "Please enter a valid email address",
                    passwordRequired: "Please enter your password",
                    passwordLength: "Password must be at least 7 characters",
                },
            },
        },
        common: {
            copy: {
                success: "Copied successfully",
            },
            saving: "Saving...",
            logout: "Sign out",
            settings: "Settings",
            defaultEmail: "administrator@example.com",
            http: {
                notLoggedIn: "You are not logged in",
                loginExpired: "Your session has expired",
                noPermission: "You do not have permission",
                notFound: "Not found",
                unknownError: "Unknown error",
                unknownException: "An unknown exception occurred",
            },
        },
    },
    "vi-VN": {
        auth: {
            signIn: {
                title: "Đăng nhập",
                description: "Nhập email và mật khẩu để đăng nhập",
                email: "Email",
                emailPlaceholder: "name@example.com",
                password: "Mật khẩu",
                passwordPlaceholder: "Nhập mật khẩu của bạn",
                forgotPassword: "Quên mật khẩu?",
                submit: "Đăng nhập",
                rememberMe: "Ghi nhớ tôi",
                resetPassword: {
                    title: "Đặt lại mật khẩu",
                    description: "Chạy lệnh sau trong thư mục gốc của trang web để đặt lại mật khẩu",
                    command: "php artisan reset:password admin-email",
                },
                validation: {
                    emailRequired: "Vui lòng nhập địa chỉ email",
                    emailInvalid: "Vui lòng nhập địa chỉ email hợp lệ",
                    passwordRequired: "Vui lòng nhập mật khẩu",
                    passwordLength: "Mật khẩu phải có ít nhất 7 ký tự",
                },
            },
        },
        common: {
            copy: {
                success: "Đã sao chép thành công",
            },
            saving: "Đang lưu...",
            logout: "Đăng xuất",
            settings: "Cài đặt",
            defaultEmail: "administrator@example.com",
            http: {
                notLoggedIn: "Bạn chưa đăng nhập",
                loginExpired: "Phiên đăng nhập đã hết hạn",
                noPermission: "Bạn không có quyền truy cập",
                notFound: "Không tìm thấy",
                unknownError: "Lỗi không xác định",
                unknownException: "Đã xảy ra ngoại lệ không xác định",
            },
        },
    },
    "zh-CN": {
        auth: {
            signIn: {
                title: "登录",
                description: "请输入您的邮箱和密码登录系统",
                email: "邮箱地址",
                emailPlaceholder: "name@example.com",
                password: "密码",
                passwordPlaceholder: "请输入密码",
                forgotPassword: "忘记密码？",
                submit: "登录",
                rememberMe: "记住我",
                resetPassword: {
                    title: "重置密码",
                    description: "在站点目录下执行以下命令找回密码",
                    command: "php artisan reset:password 管理员邮箱",
                },
                validation: {
                    emailRequired: "请输入邮箱地址",
                    emailInvalid: "邮箱地址格式不正确",
                    passwordRequired: "请输入密码",
                    passwordLength: "密码长度至少为7个字符",
                },
            },
        },
        common: {
            copy: {
                success: "复制成功",
            },
            saving: "保存中...",
            logout: "退出登录",
            settings: "设置",
            defaultEmail: "administrator@example.com",
            http: {
                notLoggedIn: "您尚未登录",
                loginExpired: "登录已过期",
                noPermission: "没有权限",
                notFound: "未找到",
                unknownError: "未知错误",
                unknownException: "发生未知异常",
            },
        },
    },
};

const localeLoaders = {
    "en-US": () => import("./locales/en-US.js"),
    "vi-VN": () => import("./locales/vi-VN.js"),
    "zh-CN": () => import("./locales/zh-CN.js"),
};

function getWindowTranslations(language) {
    if (typeof window === "undefined") {
        return null;
    }

    return window.XBOARD_TRANSLATIONS?.[language] || null;
}

function isObject(value) {
    return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function mergeDeep(base, extra) {
    if (!isObject(base) || !isObject(extra)) {
        return extra ?? base;
    }

    const result = { ...base };
    for (const [key, value] of Object.entries(extra)) {
        result[key] = isObject(value) ? mergeDeep(base[key], value) : value;
    }

    return result;
}

function getFallbackLanguage(language) {
    return LANGUAGE_OPTIONS.some((item) => item.code === language) ? language : DEFAULT_LANGUAGE;
}

function getStoredLanguage() {
    if (typeof window === "undefined") {
        return DEFAULT_LANGUAGE;
    }

    const queryLanguage = new URLSearchParams(window.location.search).get("lang");
    if (LANGUAGE_OPTIONS.some((item) => item.code === queryLanguage)) {
        return queryLanguage;
    }

    const storedLanguage = window.localStorage.getItem("i18nextLng");
    if (LANGUAGE_OPTIONS.some((item) => item.code === storedLanguage)) {
        return storedLanguage;
    }

    const navigatorLanguage = window.navigator?.language;
    if (LANGUAGE_OPTIONS.some((item) => item.code === navigatorLanguage)) {
        return navigatorLanguage;
    }

    return DEFAULT_LANGUAGE;
}

function getMessagesForLanguage(language) {
    const fallbackLanguage = getFallbackLanguage(language);
    const windowMessages = getWindowTranslations(fallbackLanguage);
    const fallbackMessages = FALLBACK_LOCALES[fallbackLanguage] || FALLBACK_LOCALES[DEFAULT_LANGUAGE];
    return mergeDeep(fallbackMessages, windowMessages || {});
}

function readPath(object, path) {
    return path.split(".").reduce((accumulator, segment) => {
        if (!accumulator || typeof accumulator !== "object") {
            return undefined;
        }

        return accumulator[segment];
    }, object);
}

function interpolate(message, values) {
    if (!isObject(values)) {
        return message;
    }

    return message.replace(/\{\{\s*([\w.-]+)\s*\}\}/g, (match, variableName) => {
        const value = readPath(values, variableName);
        return value == null ? match : String(value);
    });
}

function translate(messages, namespace, key, fallbackOrValues, explicitFallback) {
    const [explicitNamespace, explicitKey] = key.includes(":") ? key.split(":") : [namespace, key];
    const value = readPath(messages?.[explicitNamespace], explicitKey);
    const values = isObject(fallbackOrValues) ? fallbackOrValues : null;
    const fallback = isObject(fallbackOrValues) ? explicitFallback : fallbackOrValues;

    if (typeof value === "string") {
        return interpolate(value, values);
    }

    return interpolate(fallback || key, values);
}

const I18nContext = createContext({
    language: DEFAULT_LANGUAGE,
    messages: FALLBACK_LOCALES[DEFAULT_LANGUAGE],
    languages: LANGUAGE_OPTIONS,
    changeLanguage: () => {},
    t: (key) => key,
});

export function I18nProvider({ children }) {
    const [language, setLanguage] = useState(getStoredLanguage);
    const [messages, setMessages] = useState(() => getMessagesForLanguage(getStoredLanguage()));

    useEffect(() => {
        let active = true;

        const loader = localeLoaders[language];
        setMessages(getMessagesForLanguage(language));

        if (loader) {
            loader()
                .then((module) => {
                    if (!active) {
                        return;
                    }

                    const localeMessages = module?.default || module;
                    setMessages(mergeDeep(getMessagesForLanguage(language), localeMessages || {}));
                })
                .catch(() => {
                    if (active) {
                        setMessages(getMessagesForLanguage(language));
                    }
                });
        }

        if (typeof document !== "undefined") {
            document.documentElement.lang = language;
        }

        if (typeof window !== "undefined") {
            window.localStorage.setItem("i18nextLng", language);
        }

        return () => {
            active = false;
        };
    }, [language]);

    const value = useMemo(() => {
        const changeLanguage = (nextLanguage) => {
            const validLanguage = getFallbackLanguage(nextLanguage);
            if (typeof window !== "undefined") {
                window.localStorage.setItem("i18nextLng", validLanguage);
            }
            setLanguage(validLanguage);
        };

        const t = (key, valuesOrFallback, fallback) => translate(messages, "auth", key, valuesOrFallback, fallback);

        return {
            language,
            messages,
            languages: LANGUAGE_OPTIONS,
            changeLanguage,
            t,
        };
    }, [language, messages]);

    return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useTranslation(namespace = "auth") {
    const context = useContext(I18nContext);

    return useMemo(() => {
        const t = (key, valuesOrFallback, fallback) => translate(context.messages, namespace, key, valuesOrFallback, fallback);

        return {
            language: context.language,
            languages: context.languages,
            changeLanguage: context.changeLanguage,
            t,
        };
    }, [context, namespace]);
}
