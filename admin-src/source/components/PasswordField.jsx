import React, { useState } from "react";
import { classNames } from "../utils/classNames.js";
import { Button } from "./Button.jsx";
import { Input } from "./Input.jsx";

export const PasswordField = React.forwardRef(function PasswordField({ className, ...props }, ref) {
    const [visible, setVisible] = useState(false);

    return (
        <div className={classNames("relative rounded-md", className)}>
            <Input
                ref={ref}
                type={visible ? "text" : "password"}
                className="pr-16"
                {...props}
            />
            <Button
                type="button"
                variant="ghost"
                size="sm"
                className="absolute right-1 top-1/2 h-6 -translate-y-1/2 rounded-md px-2 text-xs text-muted-foreground"
                onClick={() => setVisible((current) => !current)}
                aria-label={visible ? "Hide password" : "Show password"}
            >
                {visible ? "Hide" : "Show"}
            </Button>
        </div>
    );
});
