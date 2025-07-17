import { SearchBoxForm } from "./SearchBoxForm";
import { SearchPageForm } from "./SearchPageForm";

export const EditForm = memo(props => {
  const { record } = props;
  const [type, setType] = useState('searchbox');

  useEffect(() => {
    setType(record?.type === 'searchpage' ? 'searchpage': 'searchbox')
  }, [record])

  return (
    type === 'searchbox' ? (
      <SearchBoxForm {...props} type={type} setType={setType}/>
    ) : <SearchPageForm {...props} type={type} setType={setType}/>
  )
});
