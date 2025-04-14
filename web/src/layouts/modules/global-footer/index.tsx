import DarkModeContainer from '@/components/stateless/common/DarkModeContainer';

const GlobalFooter = memo(() => {
  const currentYear = new Date().getFullYear();

  return (
    <DarkModeContainer className="h-full flex-center">
      <a
        href="https://github.com/infinilabs"
        rel="noopener noreferrer"
        target="_blank"
      >
        Copyright MIT © {currentYear} INFINI Labs
      </a>
    </DarkModeContainer>
  );
});

export default GlobalFooter;
