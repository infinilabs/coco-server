import { createLucideIcon } from "lucide-react";

const RefreshIcon = createLucideIcon('RefreshIcon', [
  ['path', { d: 'M42 8V24', transform: 'scale(0.5)', strokeWidth: '4', key: 'arrow-top' }],
  ['path', { d: 'M6 24L6 40', transform: 'scale(0.5)', strokeWidth: '4', key: 'arrow-bottom' }],
  ['path', { d: 'M6 24C6 33.9411 14.0589 42 24 42C28.8556 42 33.2622 40.0774 36.5 36.9519', transform: 'scale(0.5)', strokeWidth: '4', key: 'arc-bottom' }],
  ['path', { d: 'M42.0007 24C42.0007 14.0589 33.9418 6 24.0007 6C18.9152 6 14.3223 8.10896 11.0488 11.5', transform: 'scale(0.5)', strokeWidth: '4', key: 'arc-top' }],
]);

export default RefreshIcon;