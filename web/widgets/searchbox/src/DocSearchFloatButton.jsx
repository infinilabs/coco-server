import { Logo } from "./icons/Logo";

export const DocSearchFloatButton = ({
  onClick,
  translations = {},
}) => {
  const { buttonAriaLabel = "Search whatever you want..." } = translations;

  return (
    <button
      type="button"
      className="infini__searchbox-float-btn"
      onClick={onClick}
      aria-label={buttonAriaLabel}
    >
      <span className="infini__searchbox-float-btn-container">
        <Logo className="infini__searchbox-float-btn-icon" />
      </span>
    </button>
  );
};
