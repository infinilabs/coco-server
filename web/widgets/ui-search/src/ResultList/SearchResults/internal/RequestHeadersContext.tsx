import { createContext } from "react";

export const RequestHeadersContext = createContext<
  Record<string, string> | undefined
>(undefined);
