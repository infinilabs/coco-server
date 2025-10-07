import type { SVGProps } from 'react';
import logo from '@/assets/svg-icon/logo.svg';

const SystemLogo = memo((props: SVGProps<SVGSVGElement>) => {
  return <img {...props} src={logo} />
});

export default SystemLogo;
