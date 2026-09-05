import { useState } from "react";
import { extractAuthToken, extractUserInfo, fetchUserInfo, signIn } from "../api/auth.js";
import { useTranslation } from "../i18n/index.js";
import { navigate } from "../router/index.js";
import { useQueryParam } from "./useQueryParam.js";
import { getErrorMessage } from "../utils/errors.js";
import { setAuthToken, setUserInfo } from "../store/auth.js";

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

function validateValues(values, t) {
    const errors = {};
    const email = values.email.trim();

    if (!email) {
        errors.email = t("signIn.validation.emailRequired");
    } else if (!EMAIL_PATTERN.test(email)) {
        errors.email = t("signIn.validation.emailInvalid");
    }

    if (!values.password) {
        errors.password = t("signIn.validation.passwordRequired");
    } else if (values.password.length < 7) {
        errors.password = t("signIn.validation.passwordLength");
    }

    return errors;
}

export function useSignInPage() {
    const { t } = useTranslation("auth");
    const redirectTarget = useQueryParam("redirect", "/");
    const [values, setValues] = useState({
        email: "",
        password: "",
    });
    const [fieldErrors, setFieldErrors] = useState({});
    const [submitError, setSubmitError] = useState("");
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [isResetOpen, setIsResetOpen] = useState(false);

    function handleChange(event) {
        const { name, value } = event.target;
        setValues((current) => ({
            ...current,
            [name]: value,
        }));

        setFieldErrors((current) => {
            if (!current[name]) {
                return current;
            }

            const nextErrors = { ...current };
            delete nextErrors[name];
            return nextErrors;
        });

        if (submitError) {
            setSubmitError("");
        }
    }

    function openResetDialog() {
        setIsResetOpen(true);
    }

    function closeResetDialog() {
        setIsResetOpen(false);
    }

    async function handleSubmit(event) {
        event.preventDefault();

        const normalizedValues = {
            email: values.email.trim(),
            password: values.password,
        };

        const nextErrors = validateValues(normalizedValues, t);
        if (Object.keys(nextErrors).length) {
            setFieldErrors(nextErrors);
            return;
        }

        setIsSubmitting(true);
        setSubmitError("");

        try {
            const response = await signIn(normalizedValues);
            const authToken = extractAuthToken(response);

            if (!authToken) {
                throw new Error(response?.message || t("common:http.unknownError"));
            }

            setAuthToken(authToken);

            const userResponse = await fetchUserInfo();
            setUserInfo(extractUserInfo(userResponse));

            navigate(redirectTarget || "/");
        } catch (error) {
            setSubmitError(getErrorMessage(error, t("common:http.unknownError")));
        } finally {
            setIsSubmitting(false);
        }
    }

    return {
        values,
        fieldErrors,
        submitError,
        isSubmitting,
        isResetOpen,
        setIsResetOpen,
        openResetDialog,
        closeResetDialog,
        handleChange,
        handleSubmit,
        resetCommand: t("signIn.resetPassword.command"),
    };
}
