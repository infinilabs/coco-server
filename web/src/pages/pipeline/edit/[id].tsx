import { Spin } from 'antd';
import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';

import { getPipeline } from '@/service/api/pipeline';
import { PipelineEditor, type PipelineDoc } from '../components/PipelineEditor';

export function Component() {
  const { id } = useParams();
  const [pipeline, setPipeline] = useState<PipelineDoc | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!id) return;
    setLoading(true);
    getPipeline(id)
      .then((res: any) => {
        setPipeline((res?.data?._source ?? null) as PipelineDoc | null);
      })
      .finally(() => setLoading(false));
  }, [id]);

  if (loading) {
    return (
      <div className="flex justify-center py-120px">
        <Spin />
      </div>
    );
  }

  return <PipelineEditor initial={pipeline ?? undefined} />;
}
