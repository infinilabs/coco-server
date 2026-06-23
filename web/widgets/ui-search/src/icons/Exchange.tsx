import { createLucideIcon } from "lucide-react";

const ExchangeIcon = createLucideIcon('ExchangeIcon', [
  ['path', {
    d: 'M19 6L19 42',
    transform: 'scale(0.5)',
    strokeWidth: '4',
    strokeLinecap: 'round',
    strokeLinejoin: 'round',
    key: 'arrow-up-line'
  }],
  ['path', {
    d: 'M7 17.8995L19 5.89949',
    transform: 'scale(0.5)',
    strokeWidth: '4',
    strokeLinecap: 'round',
    strokeLinejoin: 'round',
    key: 'arrow-up-head'
  }],
  ['path', {
    d: 'M29 42.1005L29 6.10051',
    transform: 'scale(0.5)',
    strokeWidth: '4',
    strokeLinecap: 'round',
    strokeLinejoin: 'round',
    key: 'arrow-down-line'
  }],
  ['path', {
    d: 'M29 42.1005L41 30.1005',
    transform: 'scale(0.5)',
    strokeWidth: '4',
    strokeLinecap: 'round',
    strokeLinejoin: 'round',
    key: 'arrow-down-head'
  }],
]);

export default ExchangeIcon;