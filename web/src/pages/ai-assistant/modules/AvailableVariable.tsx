import { Flex, Popover } from 'antd';
import { Info } from 'lucide-react';
import type { FC } from 'react';

interface AvailableVariableProps {
  readonly type: 'answering_model' | 'caller_model' | 'intent_analysis_model' | 'picking_doc_model';
}

const AvailableVariable: FC<AvailableVariableProps> = props => {
  const { type } = props;
  const { t } = useTranslation();

  const variables: Record<AvailableVariableProps['type'], string[]> = {
    answering_model: [
      `{{.context}} ${t('page.assistant.labels.searchContext')}`,
      `{{.query}} ${t('page.assistant.labels.userQuery')}`
    ],
    caller_model: [],
    intent_analysis_model: [
      `{{.history}} ${t('page.assistant.labels.chatHistory')}`,
      `{{.tool_list}} ${t('page.assistant.labels.toolList')}`,
      `{{.network_sources}} ${t('page.assistant.labels.webSources')}`,
      `{{.query}} ${t('page.assistant.labels.userQuery')}`
    ],
    picking_doc_model: [
      `{{.query}} ${t('page.assistant.labels.userQuery')}`,
      `{{.intent}} ${t('page.assistant.labels.detectedIntent')}`,
      `{{.docs}} ${t('page.assistant.labels.matchedDocs')}`
    ]
  };

  return (
    variables[type].length !== 0 && (
      <Popover
        title={t('page.assistant.labels.availableVariablesDesc')}
        content={
          <Flex
            vertical
            gap={4}
          >
            {variables[type].map(variable => (
              <span key={variable}>{variable}</span>
            ))}
          </Flex>
        }
      >
        <div className='inline-flex cursor-pointer items-center gap-1 pt-1'>
          <span>{t('page.assistant.labels.availableVariables')}</span>

          <Info className='size-4' />
        </div>
      </Popover>
    )
  );
};

export default AvailableVariable;
