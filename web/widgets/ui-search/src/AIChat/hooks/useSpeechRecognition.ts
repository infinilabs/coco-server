import { useEffect, useRef, useState } from "react"
import type {
  ISpeechRecognition,
  ISpeechRecognitionEvent,
  SRConstructor,
} from "../types/speech"

type Options = {
  lang?: string
  autoRestart?: boolean
  onInterim?: (text: string) => void
  onFinal?: (text: string) => void
}

type UseSpeechRecognitionReturn = {
  supported: boolean
  listening: boolean
  start: () => void
  stop: () => void
}

export function useSpeechRecognition(options: Options = {}): UseSpeechRecognitionReturn {
  const { lang = "zh-CN", autoRestart = true, onInterim, onFinal } = options
  const [supported, setSupported] = useState(false)
  const [listening, setListening] = useState(false)
  const recognitionRef = useRef<ISpeechRecognition | null>(null)
  const listeningRef = useRef(false)

  useEffect(() => {
    const win = window as unknown as { SpeechRecognition?: SRConstructor; webkitSpeechRecognition?: SRConstructor }
    const SR = win.SpeechRecognition ?? win.webkitSpeechRecognition
    if (!SR) {
      setSupported(false)
      return
    }
    setSupported(true)
    const recog = new SR()
    recog.continuous = true
    recog.interimResults = true
    recog.lang = lang
    recog.onresult = (e: ISpeechRecognitionEvent) => {
      let finalText = ""
      let interimText = ""
      for (let i = e.resultIndex; i < e.results.length; i++) {
        const res = e.results[i]
        if (res.isFinal) {
          finalText += res[0].transcript
        } else {
          interimText = res[0].transcript
        }
      }
      if (finalText && onFinal) onFinal(finalText)
      if (interimText && onInterim) onInterim(interimText)
    }
    recog.onend = () => {
      if (autoRestart && listeningRef.current) {
        try { recog.start() } catch { void 0 }
      }
    }
    recognitionRef.current = recog
    return () => {
      try { recog.stop() } catch { void 0 }
      recognitionRef.current = null
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [lang, autoRestart])

  const start = () => {
    if (!supported || !recognitionRef.current) return
    try {
      recognitionRef.current.start()
      setListening(true)
      listeningRef.current = true
    } catch { void 0 }
  }

  const stop = () => {
    if (!recognitionRef.current) return
    try { recognitionRef.current.stop() } catch { void 0 }
    setListening(false)
    listeningRef.current = false
  }

  return { supported, listening, start, stop }
}
