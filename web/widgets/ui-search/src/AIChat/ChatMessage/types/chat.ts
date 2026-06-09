export interface ISource {
  id?: string;
  created?: string;
  updated?: string;
  status?: string;
  session_id?: string;
  type?: string;
  message?: any;
  attachments?: string[];
  title?: string;
  question?: string;
  details?: any[] | null;
  assistant_id?: string;
  assistant_item?: any;
  deep_research_chunks?: IChunkData[];
  [key: string]: any;
}

export interface IChatMessage {
  _id?: string;
  _source: ISource;
  [key: string]: any;
}

export interface IChunkData {
  session_id?: string;
  message_id?: string;
  message_type?: string;
  reply_to_message?: string;
  chunk_sequence?: number;
  chunk_type?: string;
  message_chunk?: string;
  [key: string]: any;
}
