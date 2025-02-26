import boxen, { type Options as BoxenOptions } from 'boxen';
import gradientString from 'gradient-string';
import type { Plugin } from 'vite';

const welcomeMessage = gradientString('#646cff', 'magenta').multiline(
  `Welcome to Coco Server!`
);

const boxenOptions: BoxenOptions = {
  borderColor: '#646cff',
  borderStyle: 'round',
  padding: 0.5
};

export function setupProjectInfo(): Plugin {
  return {
    buildStart() {
      console.log(boxen(welcomeMessage, boxenOptions));
    },

    name: 'vite:buildInfo'
  };
}
