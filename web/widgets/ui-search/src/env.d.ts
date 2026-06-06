declare module "*.css" {}
declare module "*.less" {
  const classes: { [key: string]: string };
  export default classes;
}
declare module "*.svg" {
  const content: string;
  export default content;
}
declare module '@infinilabs/ai-chat' {
  export const History: any;
  export const Chat: any;
  export const AssistantList: any;
  export const ChatInput: any;
}
declare module '@infinilabs/attachments' {
  export const Attachments: any;
}
declare module '@infinilabs/doc-detail' {
  export const ActionButton: any;
  export const DocDetail: any;
}
declare module '@infinilabs/search-results' {
  const SearchResults: any;
  export default SearchResults;
}
