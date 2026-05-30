import ListContainer from "./ListContainer";

export const SUGGESTION_FILTER_FIELDS = "field_names";

export default function FilterFields(props) {

  return (
    <ListContainer
      type={SUGGESTION_FILTER_FIELDS}
      title="过滤条件"
      {...props}
    />
  )
}