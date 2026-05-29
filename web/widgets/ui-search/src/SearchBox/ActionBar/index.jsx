import SearchActions from "./SearchActions";
import Operations from "./Operations";

export default function ActionBar({
  action_type,
  search_type,
  onSearchTypeChange,
  onSearchActionClick,
  onSearchActionDropdownClose,
  attachments,
  onAttachmentsChange,
  onSearch,
  searchable,
  className = "",
}) {
  return (
    <div className={`flex justify-between items-center px-12px ${className}`}>
      <SearchActions
        actionType={action_type}
        searchType={search_type}
        onSearchTypeChange={onSearchTypeChange}
        onButtonClick={onSearchActionClick}
        onDropdownClose={onSearchActionDropdownClose}
      />
      <Operations
        attachments={attachments}
        setAttachments={onAttachmentsChange}
        onSearch={onSearch}
        disabled={!searchable}
      />
    </div>
  );
}
