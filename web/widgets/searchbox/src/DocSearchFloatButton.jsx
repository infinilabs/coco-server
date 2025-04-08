import { Logo } from './icons/Logo';

export const DocSearchFloatButton = ({ onClick, translations = {} }) => {
  const { buttonAriaLabel = 'Search whatever you want...' } = translations;

  return (
    <button
      aria-label={buttonAriaLabel}
      className="infini__searchbox-float-btn"
      type="button"
      onClick={onClick}
    >
      <span className="infini__searchbox-float-btn-container">
        <Logo className="infini__searchbox-float-btn-icon" />
      </span>
    </button>
  );
};
