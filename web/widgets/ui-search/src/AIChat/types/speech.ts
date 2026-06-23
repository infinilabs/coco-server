// 最小语音识别类型定义，供 hooks/组件复用，避免 any
export type ISpeechRecognitionAlternative = { transcript: string }
export type SRConstructor = new () => ISpeechRecognition
export interface ISpeechRecognitionResult {
  isFinal: boolean
  0: ISpeechRecognitionAlternative
  length: number
}
export interface ISpeechRecognitionEvent {
  resultIndex: number
  results: ISpeechRecognitionResult[]
}
export interface ISpeechRecognition {
  start(): void
  stop(): void
  continuous: boolean
  interimResults: boolean
  lang: string
  onresult: ((event: ISpeechRecognitionEvent) => void) | null
  onend: (() => void) | null
}
