import { Logo } from "./icons/Logo";

export const DocSearchFloatButton = ({
  onClick,
  translations = {},
}) => {
  const { buttonAriaLabel = "Search whatever you want..." } = translations;

  return (
    <button
      type="button"
      className="docsearch-float-btn"
      onClick={onClick}
      aria-label={buttonAriaLabel}
    >
      <span className="docsearch-float-btn-container">
        <Logo className="docsearch-float-btn-icon" />
      </span>
    </button>
  );
};
