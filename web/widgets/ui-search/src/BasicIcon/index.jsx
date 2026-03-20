import { useState } from 'react';

export const AsyncLucideIcon = ({
  className,
  iconKey,
  ...props
}) => {
  const [IconComponent, setIconComponent] = useState(null);

  useEffect(() => {
    setIconComponent(null);

    const loadIcon = async () => {
      if (!iconKey) return;

      try {
        const pascalCaseKey = iconKey
          .split(/[-_]/)
          .map(part => part.charAt(0).toUpperCase() + part.slice(1))
          .join('');

        const lucideModule = await import('lucide-react');
        const Icon = lucideModule[pascalCaseKey];

        if (Icon) {
          setIconComponent(Icon);
        } else {
          console.error(`Lucide 中无 "${pascalCaseKey}" 图标（key: ${iconKey}）`);
        }
      } catch (error) {
        console.error('动态加载 Lucide 图标失败：', error);
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

export default (props) => {
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