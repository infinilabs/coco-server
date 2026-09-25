import type { ReactNode, FC } from 'react';
import type { LucideProps } from 'lucide-react';

import { DefaultLucideIcon, lookupLucideIcon } from './lucideIcons';

interface AsyncLucideIconProps extends LucideProps {
  className?: string;
  iconKey?: string;
}

// Synchronous since the registry switch: the old dynamic barrel import made
// every lucide icon load-lazy (a blank first frame) while dragging the whole
// library into the bundle.
export const AsyncLucideIcon: FC<AsyncLucideIconProps> = ({
  className,
  iconKey,
  ...props
}) => {
  const IconComponent = (iconKey && lookupLucideIcon(iconKey)) || DefaultLucideIcon;

  return (
    <div className={className}>
      <IconComponent {...props} className='w-full h-full' strokeWidth={1} />
    </div>
  )
}

interface BasicIconProps {
  className?: string;
  icon?: string | ReactNode;
}

const BasicIcon: FC<BasicIconProps> = (props) => {
  const { className = '', icon = '' } = props;

  return typeof icon === 'string' ? (
    icon?.startsWith('http') || icon?.startsWith('data:') ? (
      <div className={className}>
        <img src={icon} className='w-full h-full' />
      </div>
    ) : (
      <AsyncLucideIcon className={className} iconKey={icon} />
    )
  ) : <div className={className}>{icon}</div>
}

export default BasicIcon;
