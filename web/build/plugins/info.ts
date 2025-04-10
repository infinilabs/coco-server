import boxen, { type Options as BoxenOptions } from 'boxen';
import gradientString from 'gradient-string';
import type { Plugin } from 'vite';

const welcomeMessage = gradientString('#0087FF', 'magenta').multiline(`Welcome to Coco Server!`);

const boxenOptions: BoxenOptions = {
  borderColor: '#0087FF',
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
