import { memo, type FC } from "react";
import SearchActions from "./SearchActions";
import Operations from "./Operations";

interface ActionBarProps {
  action_type?: string;
  search_type?: string;
  onSearchTypeChange?: (type: string) => void;
  onSearchActionClick?: () => void;
  onSearchActionDropdownClose?: () => void;
  attachments?: any[];
  onAttachmentsChange?: (updater: (list: any[]) => any[]) => void;
  onSearch?: () => void;
  searchable?: boolean;
  className?: string;
  onAttachmentUpload?: (files: File[], cb: (res: any) => void) => void;
}

const ActionBar: FC<ActionBarProps> = ({
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
  onAttachmentUpload
}) => {
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
        onAttachmentUpload={onAttachmentUpload}
        action_type={action_type}
      />
    </div>
  );
}

export default memo(ActionBar);
