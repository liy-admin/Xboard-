import React, { cloneElement, isValidElement, useState } from "react";

export function TooltipProvider({ children }) {
    return <>{children}</>;
}

export function Tooltip({ children }) {
    const [open, setOpen] = useState(false);
    const enhancedChildren = React.Children.map(children, (child) => {
        if (!isValidElement(child)) {
            return child;
        }

        if (child.type === TooltipTrigger) {
            return cloneElement(child, { open, setOpen });
        }

        if (child.type === TooltipContent) {
            return cloneElement(child, { open });
        }

        return child;
    });

    return <>{enhancedChildren}</>;
}

export function TooltipTrigger({ children, asChild, setOpen }) {
    const triggerProps = {
        onMouseEnter: () => setOpen?.(true),
        onMouseLeave: () => setOpen?.(false),
        onFocus: () => setOpen?.(true),
        onBlur: () => setOpen?.(false),
    };

    if (asChild && isValidElement(children)) {
        return cloneElement(children, {
            ...triggerProps,
            onMouseEnter: (event) => {
                children.props.onMouseEnter?.(event);
                triggerProps.onMouseEnter(event);
            },
            onMouseLeave: (event) => {
                children.props.onMouseLeave?.(event);
                triggerProps.onMouseLeave(event);
            },
            onFocus: (event) => {
                children.props.onFocus?.(event);
                triggerProps.onFocus(event);
            },
            onBlur: (event) => {
                children.props.onBlur?.(event);
                triggerProps.onBlur(event);
            },
        });
    }

    return <span {...triggerProps}>{children}</span>;
}

export function TooltipContent({ children, open }) {
    if (!open) {
        return null;
    }

    return (
        <div className="tooltip-content" role="tooltip">
            {children}
        </div>
    );
}
