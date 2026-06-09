export interface ChatMessageSource {
  type: string;
  assistant_id?: string;
  message: string;
  question?: string;
  [key: string]: unknown;
}

export interface ChatMessageItem {
  _id: string;
  _source: ChatMessageSource;
}

export interface Chat {
  _id: string;
  messages?: ChatMessageItem[];
  _source?: {
    id?: string;
    [key: string]: unknown;
  };
}

export interface IChunkData {
  message_chunk?: string;
  [key: string]: unknown;
}
