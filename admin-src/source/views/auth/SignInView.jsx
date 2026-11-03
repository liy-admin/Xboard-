import React from "react";
import { BRAND_NAME, BrandLockup } from "../../components/Brand.jsx";
import { Button } from "../../components/Button.jsx";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../components/Card.jsx";
import { Input } from "../../components/Input.jsx";
import { LanguageSwitcher } from "../../components/LanguageSwitcher.jsx";
import { PasswordField } from "../../components/PasswordField.jsx";
import { useDocumentTitle } from "../../composables/useDocumentTitle.js";
import { useSignInPage } from "../../composables/useSignInPage.js";
import { useTranslation } from "../../i18n/index.js";
import { ResetPasswordDialog } from "./ResetPasswordDialog.jsx";

export function SignInView() {
    const { t } = useTranslation("auth");
    const page = useSignInPage();
    const appTitle = globalThis.window?.settings?.title || BRAND_NAME;
    const appDescription = globalThis.window?.settings?.description || "";

    useDocumentTitle(`${appTitle} · ${t("signIn.title")}`);

    return (
        <>
            <div className="lv-auth">
                <aside className="lv-auth__hero">
                    <BrandLockup size={44} />
                    <h2 className="lv-auth__headline">{t("signIn.heroTitle")}</h2>
                    <p className="lv-auth__sub">{appDescription || t("signIn.heroSubtitle")}</p>
                    <ul className="lv-auth__points">
                        <li>{t("signIn.heroPoint1")}</li>
                        <li>{t("signIn.heroPoint2")}</li>
                        <li>{t("signIn.heroPoint3")}</li>
                    </ul>
                </aside>

                <section className="lv-auth__panel">
                    <div className="absolute right-4 top-4 md:right-8 md:top-8">
                        <LanguageSwitcher />
                    </div>

                    <div className="lv-auth__form">
                        <div className="lv-auth__mobile-brand">
                            <BrandLockup size={40} />
                        </div>

                    <Card className="p-4 sm:p-6">
                        <CardHeader className="p-0 pb-4">
                            <CardTitle className="text-xl font-semibold tracking-tight sm:text-2xl">
                                {t("signIn.title")}
                            </CardTitle>
                            <CardDescription>{t("signIn.description")}</CardDescription>
                        </CardHeader>

                        <CardContent className="p-0">
                            <form className="space-y-4" onSubmit={page.handleSubmit}>
                                {page.submitError ? (
                                    <div className="text-sm text-destructive">{page.submitError}</div>
                                ) : null}

                                <div className="space-y-2">
                                    <label className="text-sm font-medium leading-none" htmlFor="sign-in-email">
                                        {t("signIn.email")}
                                    </label>
                                    <Input
                                        id="sign-in-email"
                                        name="email"
                                        type="email"
                                        autoComplete="email"
                                        placeholder={t("signIn.emailPlaceholder")}
                                        value={page.values.email}
                                        onChange={page.handleChange}
                                        aria-invalid={page.fieldErrors.email ? "true" : "false"}
                                    />
                                    {page.fieldErrors.email ? (
                                        <p className="text-sm text-destructive">{page.fieldErrors.email}</p>
                                    ) : null}
                                </div>

                                <div className="space-y-2">
                                    <label className="text-sm font-medium leading-none" htmlFor="sign-in-password">
                                        {t("signIn.password")}
                                    </label>
                                    <PasswordField
                                        id="sign-in-password"
                                        name="password"
                                        autoComplete="current-password"
                                        placeholder={t("signIn.passwordPlaceholder")}
                                        value={page.values.password}
                                        onChange={page.handleChange}
                                        aria-invalid={page.fieldErrors.password ? "true" : "false"}
                                    />
                                    {page.fieldErrors.password ? (
                                        <p className="text-sm text-destructive">{page.fieldErrors.password}</p>
                                    ) : null}
                                </div>

                                <div className="flex items-center justify-between">
                                    <Button
                                        type="button"
                                        variant="link"
                                        className="px-0 text-sm font-normal text-muted-foreground hover:text-primary"
                                        onClick={page.openResetDialog}
                                    >
                                        {t("signIn.forgotPassword")}
                                    </Button>
                                </div>

                                <Button type="submit" className="w-full" size="lg" loading={page.isSubmitting}>
                                    {t("signIn.submit")}
                                </Button>
                            </form>
                        </CardContent>
                    </Card>
                    </div>
                </section>
            </div>

            <ResetPasswordDialog
                open={page.isResetOpen}
                onOpenChange={page.setIsResetOpen}
                command={page.resetCommand}
            />
        </>
    );
}
