import { Button } from "antd";
import { useRecordHotkeys } from 'react-hotkeys-hook';

const MAPPING_TO_SYMBOL = {
    "minus": `-`,
    "equal": `=`,
    "bracketleft": `[`,
    "bracketright": `]`,
    "semicolon": `;`,
    "quote": `'`,
    "backslash": `\\`,
    "comma": `,`,
    "period": `.`,
    "slash": `/`,
}

const MAPPING_TO_NAME = {
    "-": `minus`,
    "=": `equal`,
    "[": `bracketleft`,
    "]": `bracketright`,
    ";": `semicolon`,
    "'": `quote`,
    "\\": `backslash`,
    ",": `comma`,
    ".": `period`,
    "/": `slash`,
}

export const HotKeys = memo((props) => {
    const { value, onChange, className, placeholder } = props;

    const [keys, { start, stop, isRecording }] = useRecordHotkeys()

    const handleClick = () => {
        if (!isRecording) {
            start()
        }
    }

    useEffect(() => {
       if (keys?.size >= 2) {
            stop()
            onChange(Array.from(keys).slice(0, 2).map((item) => MAPPING_TO_SYMBOL[item] || item).join('+'))
       }
    }, [keys?.size])

    useEffect(() => {
        if (isRecording) {
            window.addEventListener('keyup', stop)
            window.addEventListener('mouseup', stop)
        }
        return () => {
            window.removeEventListener('keyup', stop)
            window.removeEventListener('mouseup', stop)
        }
    }, [isRecording])

    const defultText = (<span className="text-[#c4c4c4]">{placeholder}</span>)

    return (
        <Button onClick={handleClick} className={`${className} flex justify-left px-11px ${isRecording ? 'border-[#0087ff]' : ''}`}>
            {
                isRecording ? (
                    keys?.size > 0 ? Array.from(keys).map((item) => MAPPING_TO_SYMBOL[item] || item).join('+') : defultText
                ) : (
                    value || defultText
                )
            }
        </Button>
    );
})