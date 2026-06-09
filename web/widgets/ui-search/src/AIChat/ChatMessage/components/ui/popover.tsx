import * as React from "react"
import clsx from "clsx"

const Popover = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={clsx("relative inline-block", className)}
    {...props}
  />
))
Popover.displayName = "Popover"

const PopoverTrigger = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={clsx("inline-block cursor-pointer", className)}
    {...props}
  />
))
PopoverTrigger.displayName = "PopoverTrigger"

interface PopoverContentProps extends React.HTMLAttributes<HTMLDivElement> {
  side?: "top" | "bottom" | "left" | "right"
}

const PopoverContent = React.forwardRef<
  HTMLDivElement,
  PopoverContentProps
>(({ className, side = "bottom", ...props }, ref) => (
  <div
    ref={ref}
    className={clsx(
      "absolute z-50 min-w-32 overflow-hidden rounded-md border bg-popover p-4 text-popover-foreground shadow-md outline-none",
      {
        "bottom-full mb-2": side === "top",
        "top-full mt-2": side === "bottom",
        "right-full mr-2": side === "left",
        "left-full ml-2": side === "right",
      },
      className
    )}
    {...props}
  />
))
PopoverContent.displayName = "PopoverContent"

export { Popover, PopoverTrigger, PopoverContent }
