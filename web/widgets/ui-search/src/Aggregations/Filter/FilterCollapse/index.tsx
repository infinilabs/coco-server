import {
  useState,
  type FC,
  type MouseEvent,
  type PropsWithChildren,
} from "react";
import { motion } from "motion/react";
import { BrushCleaning, SquareMinus, SquarePlus } from "lucide-react";
import { clsx } from "clsx";
import { MinusSquareOutlined, PlusSquareOutlined } from "@ant-design/icons";

export interface FilterCollapseProps extends PropsWithChildren {
  defaultExpand?: boolean;
  title: string;
  classNames?: {
    title?: string;
  };
  onClear?: (event: MouseEvent) => void;
}

const FilterCollapse: FC<FilterCollapseProps> = (props) => {
  const { defaultExpand, title, children, classNames, onClear } = props;
  const [expand, setExpand] = useState(defaultExpand ?? false);

  const toggleExpand = () => {
    setExpand((prev) => !prev);
  };

  const handleClear = (event: MouseEvent) => {
    event.stopPropagation();

    onClear?.(event);
  };

  return (
    <div>
      <div
        className={clsx(
          "flex items-center justify-between cursor-pointer text-[#999] dark:text-[#666]",
          classNames?.title
        )}
        onClick={toggleExpand}
      >
        <div className="flex items-center gap-2 text-12px">
          <div className="relative size-4 [&>*]:(absolute inset-0)">
            <motion.div
              initial={{ display: 'block' }}
              animate={{ display: expand ? 'none' : 'block' }}
            >
              <PlusSquareOutlined className="text-[#999] dark:text-[#666] text-16px"/>
            </motion.div>

            <motion.div
              initial={{ display: 'none' }}
              animate={{ display: expand ? 'block' : 'none' }}
            >
              <MinusSquareOutlined className="text-[#999] dark:text-[#666] text-16px" />
            </motion.div>
          </div>

          <span>{title}</span>
        </div>

        <BrushCleaning size={14} onClick={handleClear} />
      </div>

      <motion.div
        initial={{
          height: 0,
          opacity: 0,
        }}
        animate={{
          height: expand ? "auto" : 0,
          opacity: expand ? 1 : 0,
        }}
        className="overflow-hidden"
      >
        <div className="pt-4">{children}</div>
      </motion.div>
    </div>
  );
};

export default FilterCollapse;
