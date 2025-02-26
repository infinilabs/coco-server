import Logo from '@/assets/svg-icon/coco.svg'
import { ReactSVG } from 'react-svg';

const SystemLogoShort = () => {
  return (
    <div>
      <ReactSVG src={Logo} className='w-48px h-48px text-48px'/>
    </div>
  );
};

export default SystemLogoShort;
