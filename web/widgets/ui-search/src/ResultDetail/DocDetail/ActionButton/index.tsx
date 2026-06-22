import { Button, type ButtonProps } from "antd";
import { useState, useMemo, type FC, type MouseEvent } from "react";
import { motion } from "motion/react";

export type ActionButtonProps = ButtonProps;

const canHover = window.matchMedia("(hover: hover)").matches;

const ActionButton: FC<ActionButtonProps> = (props) => {
  const { icon, children, onMouseOver, onMouseOut, className = '', ...rest } = props;

  const [hovered, setHovered] = useState(false);

  const expanded = useMemo(() => !canHover || hovered, [hovered]);

  const handleMouseOver = (event: MouseEvent<HTMLButtonElement>) => {
    setHovered(true);

    onMouseOver?.(event);
  };

  const handleMouseOut = (event: MouseEvent<HTMLButtonElement>) => {
    setHovered(false);

    onMouseOut?.(event);
  };

  return (
    <Button
      {...rest}
      color="primary"
      variant="filled"
      shape="round"
      className={`gap-0 ${className}`}
      onMouseOver={handleMouseOver}
      onMouseOut={handleMouseOut}
    >
      <span className="inline-flex items-center children:size-4">{icon}</span>

      <motion.span
        className="overflow-hidden"
        initial={false}
        animate={{
          width: expanded ? "auto" : 0,
          opacity: Number(expanded),
          paddingLeft: Number(expanded) * 8,
        }}
      >
        {children}
      </motion.span>
    </Button>
  );
};

export default ActionButton;
