/**
 * Wire-format chunk from the /_poll and /_history endpoints.
 */
export interface IChunk {
  type: string;
  text?: string;
  tool_name?: string;
  tool_id?: string;
  seq?: number;
}

/**
 * Render-state items driven directly by chunks.
 * Each item corresponds to one visible element in the chat UI.
 */
export type UserMsg = { type: "user"; text: string };
export type AssistantMsg = { type: "assistant"; text: string };
export type QueryIntentMsg = { type: "query_intent"; text: string };
export type FetchSourceMsg = { type: "fetch_source"; text: string };
export type PickSourceMsg = { type: "pick_source"; text: string };
export type DeepReadMsg = { type: "deep_read"; text: string };
export type ToolCallMsg = {
  type: "tool_call";
  toolName: string;
  toolId: string;
  args: string;
  result: string;
};
export type DeepResearchMsg = { type: "deep_research"; chunks: IChunk[] };
export type PayloadMsg = { type: "payload"; data: unknown };

export type ChatItem =
  | UserMsg
  | AssistantMsg
  | QueryIntentMsg
  | FetchSourceMsg
  | PickSourceMsg
  | DeepReadMsg
  | ToolCallMsg
  | DeepResearchMsg
  | PayloadMsg;

/**
 * Session metadata (from the history list endpoint).
 */
export interface Session {
  _id: string;
  _source?: {
    id?: string;
    title?: string;
    [key: string]: unknown;
  };
}
