import { Button } from "antd";
import { Search } from "lucide-react";
import { CornerDownLeft } from "lucide-react";
import { Mic } from "lucide-react";
import { useEffect, useRef, useState } from "react";

export function SearchButton(props) {

    const { placeholder } = props;

    const resizeObserverRef = useRef(null)
    const btnRef = useRef(null)
    const [displayState, setDisplayState] = useState({
      key: true,
      placeholder: true,
    });

    useEffect(() => {
      if (resizeObserverRef.current) {
        resizeObserverRef.current.disconnect();
      }
      const element = btnRef.current;
      if (element) {
        resizeObserverRef.current = new ResizeObserver((entries) => {
          for (const entry of entries) {
            if (entry.target.offsetWidth < 48) {
              setDisplayState({
                key: false,
                placeholder: false,
              })
            } else if (entry.target.offsetWidth < 120) {
              setDisplayState({
                key: false,
                placeholder: true,
              })
            } else {
              setDisplayState({
                key: true,
                placeholder: true,
              })
            }
          }
        });
  
        resizeObserverRef.current.observe(element);
  
        return () => {
          resizeObserverRef.current.disconnect();
        };
      }
    }, [placeholder, btnRef.current]);

    return (
        <Button ref={btnRef} size="large" className="min-w-42px max-w-100% flex items-center justify-between px-12px">
            <Search className="w-16px h-16px"/>
            { displayState.placeholder && <div className="text-left truncate w-[calc(100%-64px)]">{placeholder}</div>}
            { displayState.key && (
              <div className="flex gap-12px items-center">
                <Mic className="w-16px h-16px"/>
                <Button className="w-20px h-20px rounded-50% p-0" type="primary">
                  <CornerDownLeft className="w-10px h-10px"/>
                </Button>
              </div>
            )}
        </Button>
    )
}

export default SearchButton;
