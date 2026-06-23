import { useEffect, useState, type ReactNode, type FC } from 'react';
import type { LucideProps } from 'lucide-react';

interface AsyncLucideIconProps extends LucideProps {
  className?: string;
  iconKey?: string;
}

export const AsyncLucideIcon: FC<AsyncLucideIconProps> = ({
  className,
  iconKey,
  ...props
}) => {
  const [IconComponent, setIconComponent] = useState<FC<LucideProps> | null>(null);

  useEffect(() => {
    setIconComponent(null);

    const loadIcon = async () => {
      if (!iconKey) return;

      try {
        const pascalCaseKey = iconKey
          .split(/[-_]/)
          .map(part => part.charAt(0).toUpperCase() + part.slice(1))
          .join('');

        /* @vite-ignore */  
        const lucideModule = await import('lucide-react');
        const Icon = (lucideModule as any)[pascalCaseKey];

        if (Icon) {
          setIconComponent(() => Icon);
        }
      } catch (error) {
      }
    };

    loadIcon();
  }, [iconKey]);

  if (!IconComponent) {
    return null
  };

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