import { FloatButton } from "antd";
import { ArrowUpToLine } from "lucide-react";
import { useCallback, useEffect, useState, type FC } from "react";

interface BackTopButtonProps {
  getContainer?: () => HTMLElement | null;
  loading?: boolean;
  zIndex?: number;
}

const BackTopButton: FC<BackTopButtonProps> = ({ getContainer, loading, zIndex = 999 }) => {
  const scrollContainer = getContainer?.() ?? null;
  const [backTopShow, setBackTopShow] = useState(false);

  const handleContainerScroll = useCallback(() => {
    if (!scrollContainer || loading) return;
    setBackTopShow(scrollContainer.scrollTop > 400);
  }, [scrollContainer, loading]);

  useEffect(() => {
    if (!scrollContainer) {
      setBackTopShow(false);
      return;
    }

    scrollContainer.addEventListener("scroll", handleContainerScroll);
    handleContainerScroll();

    return () => {
      scrollContainer.removeEventListener("scroll", handleContainerScroll);
    };
  }, [scrollContainer, handleContainerScroll]);

  const handleBackTopClick = useCallback(() => {
    if (!scrollContainer || loading) return;
    scrollContainer.scrollTo({
      top: 0,
      behavior: "smooth",
    });
  }, [scrollContainer, loading]);

  if (!scrollContainer || !backTopShow || loading) return null;

  return (
    <FloatButton.BackTop
      target={() => scrollContainer}
      visibilityHeight={0}
      duration={300}
      onClick={handleBackTopClick}
      className="!border-[#F0F0F0] !dark:border-[#303030]"
      icon={<ArrowUpToLine className="size-18px" />}
      style={{
        right: 24,
        bottom: 24,
        zIndex,
        display: "flex",
      }}
    />
  );
};

export default BackTopButton;
