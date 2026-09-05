export function getErrorMessage(error, fallback = "Something went wrong") {
    if (!error) {
        return fallback;
    }

    if (typeof error === "string") {
        return error;
    }

    if (typeof error.message === "string" && error.message.trim()) {
        return error.message;
    }

    const responseMessage = error?.response?.data?.message;
    if (typeof responseMessage === "string" && responseMessage.trim()) {
        return responseMessage;
    }

    const payloadMessage = error?.payload?.message;
    if (typeof payloadMessage === "string" && payloadMessage.trim()) {
        return payloadMessage;
    }

    const dataMessage = error?.data?.message;
    if (typeof dataMessage === "string" && dataMessage.trim()) {
        return dataMessage;
    }

    if (typeof error.payload === "string" && error.payload.trim()) {
        return error.payload;
    }

    return fallback;
}
