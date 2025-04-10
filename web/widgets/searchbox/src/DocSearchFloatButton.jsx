import { Logo } from './icons/Logo';

export const DocSearchFloatButton = ({
  settings,
  onClick,
  translations = {},
}) => {
  const { buttonAriaLabel = "Search whatever you want..." } = translations;

  return (
    <div id="infini__searchbox" data-theme={settings?.appearance?.theme}>
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
    </div>
  );
};
