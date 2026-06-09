declare module "*.css" {}
declare module "*.css?inline" {
  const css: string;
  export default css;
}
declare module "*.less" {
  const classes: { [key: string]: string };
  export default classes;
}
declare module "*.svg" {
  const content: string;
  export default content;
}

declare module '@infinilabs/attachments' {
  export const Attachments: any;
}