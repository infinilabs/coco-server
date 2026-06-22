import React from "react";
import clsx from "clsx";
import { Check } from "lucide-react";

interface CheckboxProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "onChange"> {
  indeterminate?: boolean;
  onCheckedChange?: (checked: boolean) => void;
  onChange?: (checked: boolean) => void;
  isCheckSome?: boolean;
}

const Checkbox = React.forwardRef<HTMLInputElement, CheckboxProps>(
  ({ className, indeterminate, checked, onCheckedChange, onChange, isCheckSome, ...props }, ref) => {
    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      const isChecked = e.target.checked;
      onCheckedChange?.(isChecked);
      onChange?.(isChecked);
    };

    return (
      <div className={clsx("relative inline-flex items-center justify-center", className)}>
        <input
          type="checkbox"
          ref={ref}
          className="peer h-4 w-4 appearance-none rounded-sm border border-solid border-black/30 dark:border-white/30 checked:bg-[#2F54EB] checked:border-[#2F54EB] transition cursor-pointer bg-transparent"
          checked={checked}
          onChange={handleChange}
          {...props}
        />
        {checked && (
          <Check className="absolute h-3 w-3 text-[#2F54EB] pointer-events-none" />
        )}
        {indeterminate && !checked && isCheckSome && (
          <div className="absolute h-2 w-2 bg-[#2F54EB] pointer-events-none rounded-[1px]" />
        )}
      </div>
    );
  }
);
Checkbox.displayName = "Checkbox";

export default Checkbox;
