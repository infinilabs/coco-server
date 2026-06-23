import React from 'react';

export type AppConf = {
  theme: "light" | "dark" | "auto";
  stay_on_top: boolean;
  ask_mode: boolean;
  mac_header_hidden: boolean;
};

export interface SVG extends React.SVGProps<SVGSVGElement> {
  children?: React.ReactNode;
  size?: number;
  title?: string;
  action?: boolean;
  onClick?: (e: React.MouseEvent) => void;
}
