import { SearchBoxForm } from "./SearchBoxForm";
import { FullscreenForm } from "./FullscreenForm";

export const EditForm = memo(props => {
  const { record } = props;
  const [type, setType] = useState('searchbox');

  useEffect(() => {
    setType(record?.type === 'fullscreen' ? 'fullscreen': 'searchbox')
  }, [record])

  return (
    type === 'searchbox' ? (
      <SearchBoxForm {...props} type={type} setType={setType}/>
    ) : <FullscreenForm {...props} type={type} setType={setType}/>
  )
});
