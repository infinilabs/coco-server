import Logo from './icons/logo.svg?react';

interface DocSearchFloatButtonProps {
  theme?: string;
  settings?: any;
  onClick?: () => void;
}

export const DocSearchFloatButton: React.FC<DocSearchFloatButtonProps> = ({
  theme,
  settings,
  onClick,
}) => {

  const { options } = settings || {};

  return (
    <div id="infini__searchbox" data-theme={theme}>
      <button
        type="button"
        className="infini__searchbox-float-btn"
        onClick={onClick}
      >
        <span className="infini__searchbox-float-btn-container">
          <span className="infini__searchbox-float-btn-text">{options?.floating_placeholder || 'Ask AI'}</span>
          { options?.floating_icon ? (
            <img src={options?.floating_icon}/>
          ) : <Logo /> }
        </span>
      </button>
    </div>
  );
};
