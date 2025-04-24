import { getDarkMode } from "@/store/slice/theme";

const Icon = (props) => {
    const { className = '', children } = props
    
    const darkMode = useAppSelector(getDarkMode);

    return (
        <div className={`overflow-hidden flex items-center justify-center ${className}`} style={darkMode ? { filter: `drop-shadow(0 0 6px rgba(255, 255, 255, 1))`} : {}}>
            {children}
        </div>
    )
};

export default Icon;
