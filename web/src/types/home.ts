export interface RootData {
  cluster_name: string;
  cluster_uuid: string;
  name: string;
  tagline: string;
  version: {
    build_date: string;
    build_hash: string;
    build_snapshot: boolean;
    distribution: string;
    distributor: string;
    lucene_version: string;
    minimum_lucene_index_compatibility_version: string;
    minimum_wire_lucene_version: string;
    number: string;
  };
}

// 分析器类型接口
interface AnalyzerType {
  count: number;
  index_count: number;
  name: string;
}

// 字段类型接口
interface FieldType {
  count: number;
  index_count: number;
  name: string;
}

// JVM 版本接口
interface JvmVersion {
  bundled_jdk: boolean;
  count: number;
  using_bundled_jdk: boolean | null;
  version: string;
  vm_name: string;
  vm_vendor: string;
  vm_version: string;
}

// 操作系统名称接口
interface OsName {
  count: number;
  name: string;
}

// 操作系统美化名称接口
interface OsPrettyName {
  count: number;
  pretty_name: string;
}

// 打包类型接口
interface PackagingType {
  count: number;
  flavor: string;
  type: string;
}

// 插件接口
interface Plugin {
  classname: string;
  dependency_module: string;
  description: string;
  easysearch_version: string;
  extended_plugins: string[];
  has_native_controller: boolean;
  java_version: string;
  name: string;
  version: string;
}

// 主接口
export interface ClusterStatsData {
  _nodes: {
    failed: number;
    successful: number;
    total: number;
  };
  cluster_name: string;
  cluster_uuid: string;
  indices: {
    analysis: {
      analyzer_types: AnalyzerType[];
      built_in_analyzers: AnalyzerType[];
      built_in_char_filters: AnalyzerType[];
      built_in_filters: AnalyzerType[];
      built_in_tokenizers: AnalyzerType[];
      char_filter_types: AnalyzerType[];
      filter_types: AnalyzerType[];
      tokenizer_types: AnalyzerType[];
    };
    completion: {
      size_in_bytes: number;
    };
    count: number;
    docs: {
      count: number;
      deleted: number;
    };
    fielddata: {
      evictions: number;
      memory_size_in_bytes: number;
    };
    mappings: {
      field_types: FieldType[];
    };
    query_cache: {
      cache_count: number;
      cache_size: number;
      evictions: number;
      hit_count: number;
      memory_size_in_bytes: number;
      miss_count: number;
      total_count: number;
    };
    segments: {
      count: number;
      doc_values_memory_in_bytes: number;
      file_sizes: Record<string, unknown>;
      fixed_bit_set_memory_in_bytes: number;
      index_writer_memory_in_bytes: number;
      max_unsafe_auto_id_timestamp: number;
      memory_in_bytes: number;
      norms_memory_in_bytes: number;
      points_memory_in_bytes: number;
      stored_fields_memory_in_bytes: number;
      term_vectors_memory_in_bytes: number;
      terms_memory_in_bytes: number;
      version_map_memory_in_bytes: number;
    };
    shards: {
      index: {
        primaries: {
          avg: number;
          max: number;
          min: number;
        };
        replication: {
          avg: number;
          max: number;
          min: number;
        };
        shards: {
          avg: number;
          max: number;
          min: number;
        };
      };
      primaries: number;
      replication: number;
      total: number;
    };
    store: {
      reserved_in_bytes: number;
      size_in_bytes: number;
    };
  };
  nodes: {
    count: {
      coordinating_only: number;
      data: number;
      ingest: number;
      master: number;
      remote_cluster_client: number;
      search: number;
      total: number;
    };
    discovery_types: Record<string, number>;
    fs: {
      available_in_bytes: number;
      cache_reserved_in_bytes: number;
      free_in_bytes: number;
      total_in_bytes: number;
    };
    ingest: {
      number_of_pipelines: number;
      processor_stats: Record<
        string,
        {
          count: number;
          current: number;
          failed: number;
          time_in_millis: number;
        }
      >;
    };
    jvm: {
      max_uptime_in_millis: number;
      mem: {
        heap_max_in_bytes: number;
        heap_used_in_bytes: number;
      };
      threads: number;
      versions: JvmVersion[];
    };
    network_types: {
      http_types: Record<string, number>;
      transport_types: Record<string, number>;
    };
    os: {
      allocated_processors: number;
      available_processors: number;
      mem: {
        free_in_bytes: number;
        free_percent: number;
        total_in_bytes: number;
        used_in_bytes: number;
        used_percent: number;
      };
      names: OsName[];
      pretty_names: OsPrettyName[];
    };
    packaging_types: PackagingType[];
    plugins: Plugin[];
    process: {
      cpu: {
        percent: number;
      };
      open_file_descriptors: {
        avg: number;
        max: number;
        min: number;
      };
    };
    versions: string[];
  };
  status: 'green' | 'red' | 'yellow';
  timestamp: number;
}

// 基础类型定义
export interface ClusterNode {
  attributes: Record<string, any>;
  ephemeral_id: string;
  name: string;
  transport_address: string;
}

export interface ClusterCoordination {
  last_accepted_config: string[];
  last_committed_config: string[];
  term: number;
  voting_config_exclusions: any[];
}

// 索引模板相关类型
export interface IndexTemplate {
  aliases: Record<string, any>;
  index_patterns: string[];
  mappings: {
    _doc: {
      dynamic_templates?: Array<{
        [key: string]: {
          mapping: Record<string, any>;
          match_mapping_type?: string;
          path_match?: string;
        };
      }>;
      properties?: Record<string, any>;
    };
  };
  order: number;
  settings: {
    index: Record<string, any>;
  };
}

// 索引设置类型
export interface IndexSettings {
  index: {
    analysis?: {
      analyzer: Record<
        string,
        {
          filter: string[];
          tokenizer: string;
        }
      >;
    };
    codec?: string;
    creation_date: string;
    format?: string;
    lifecycle?: {
      name: string;
      rollover_alias: string;
    };
    mapping?: {
      coerce?: string;
      ignore_malformed?: string;
      total_fields?: {
        limit: string;
      };
    };
    max_result_window?: string;
    number_of_replicas: string;
    number_of_shards: string;
    provided_name: string;
    translog?: {
      durability: string;
    };
    uuid: string;
    version: {
      created: string;
    };
  };
}

// 索引映射类型
export interface IndexMappings {
  _doc: {
    dynamic_templates?: Array<{
      [key: string]: {
        mapping: Record<string, any>;
        match_mapping_type?: string;
        path_match?: string;
      };
    }>;
    properties?: Record<string, any>;
  };
}

// 索引信息类型
export interface IndexInfo {
  aliases: Record<string, any>;
  aliases_version: number;
  in_sync_allocations?: Record<string, string[]>;
  mapping_version: number;
  mappings: IndexMappings;
  primary_terms?: Record<string, number>;
  routing_num_shards: number;
  settings: IndexSettings;
  settings_version: number;
  state: 'close' | 'open';
  version: number;
}

// 路由表类型
export interface RoutingTable {
  indices: Record<
    string,
    {
      shards: Record<
        string,
        Array<{
          allocation_id: {
            id: string;
          };
          index: string;
          node: string;
          primary: boolean;
          relocating_node?: string | null;
          shard: number;
          state: 'INITIALIZING' | 'RELOCATING' | 'STARTED' | 'UNASSIGNED';
        }>
      >;
    }
  >;
}

// 集群元数据类型
export interface ClusterMetadata {
  cluster_coordination: ClusterCoordination;
  cluster_uuid: string;
  cluster_uuid_committed: boolean;
  component_template?: Record<string, any>;
  index_graveyard?: {
    tombstones: any[];
  };
  index_template?: Record<string, any>;
  indices: Record<string, IndexInfo>;
  repository_generations?: Record<string, any>;
  templates: Record<string, IndexTemplate>;
}

// 主要的 ClusterStateData 接口
export interface ClusterStateData {
  blocks: Record<string, any>;
  cluster_name: string;
  cluster_uuid: string;
  custom?: Record<string, any>;
  master_node: string;
  metadata: ClusterMetadata;
  nodes: Record<string, ClusterNode>;
  restore?: {
    snapshots: any[];
  };
  routing_nodes?: {
    nodes: Record<string, any>;
    unassigned: any[];
  };
  routing_table?: RoutingTable;
  security_tokens?: Record<string, any>;
  snapshot_deletions?: {
    snapshot_deletions: any[];
  };
  snapshots?: {
    snapshots: any[];
  };
  state_uuid: string;
  version: number;
}

// 辅助类型：用于特定场景的简化版本
export interface ClusterStateBasic {
  cluster_name: string;
  cluster_uuid: string;
  master_node: string;
  nodes: Record<string, ClusterNode>;
  state_uuid: string;
  version: number;
}

// 索引状态枚举
export type IndexState = 'close' | 'open';

// 分片状态枚举
export type ShardState = 'INITIALIZING' | 'RELOCATING' | 'STARTED' | 'UNASSIGNED';

// 模板类型枚举
export type TemplateType = 'component_template' | 'index_template' | 'legacy_template';

export interface ClusterHealthData {
  active_primary_shards: number;
  active_shards: number;
  active_shards_percent_as_number: number;
  cluster_name: string;
  delayed_unassigned_shards: number;
  initializing_shards: number;
  number_of_data_nodes: number;
  number_of_in_flight_fetch: number;
  number_of_nodes: number;
  number_of_pending_tasks: number;
  relocating_shards: number;
  status: 'green' | 'red' | 'yellow';
  task_max_waiting_in_queue_millis: number;
  unassigned_shards: number;
}

export interface NodesStatsData {
  _nodes: {
    failed: number;
    successful: number;
    total: number;
  };
  cluster_name: string;
  nodes: {
    [nodeId: string]: {
      attributes: {
        data_tier: string;
      };
      fs: {
        data: Array<{
          available_in_bytes: number;
          cache_reserved_in_bytes: number;
          free_in_bytes: number;
          mount: string;
          path: string;
          total_in_bytes: number;
          type: string;
        }>;
        timestamp: number;
        total: {
          available_in_bytes: number;
          cache_reserved_in_bytes: number;
          free_in_bytes: number;
          total_in_bytes: number;
        };
      };
      host: string;
      http: {
        current_open: number;
        total_opened: number;
      };
      indices: {
        docs: {
          count: number;
          deleted: number;
        };
        indexing: {
          delete_current: number;
          delete_time_in_millis: number;
          delete_total: number;
          index_current: number;
          index_failed: number;
          index_time_in_millis: number;
          index_total: number;
          is_throttled: boolean;
          noop_update_total: number;
          throttle_time_in_millis: number;
        };
        search: {
          fetch_current: number;
          fetch_time_in_millis: number;
          fetch_total: number;
          open_contexts: number;
          query_current: number;
          query_time_in_millis: number;
          query_total: number;
        };
        store: {
          reserved_in_bytes: number;
          size_in_bytes: number;
        };
      };
      ip: string;
      jvm: {
        gc: {
          collectors: {
            old: {
              collection_count: number;
              collection_time_in_millis: number;
            };
            young: {
              collection_count: number;
              collection_time_in_millis: number;
            };
          };
        };
        mem: {
          heap_committed_in_bytes: number;
          heap_max_in_bytes: number;
          heap_used_in_bytes: number;
          heap_used_percent: number;
          non_heap_committed_in_bytes: number;
          non_heap_used_in_bytes: number;
        };
        threads: {
          count: number;
          peak_count: number;
        };
        timestamp: number;
        uptime_in_millis: number;
      };
      name: string;
      os: {
        cpu: {
          load_average: {
            '1m': number;
            '5m': number;
            '15m': number;
          };
          percent: number;
        };
        mem: {
          free_in_bytes: number;
          free_percent: number;
          total_in_bytes: number;
          used_in_bytes: number;
          used_percent: number;
        };
        swap: {
          free_in_bytes: number;
          total_in_bytes: number;
          used_in_bytes: number;
        };
        timestamp: number;
      };
      process: {
        cpu: {
          percent: number;
          total_in_millis: number;
        };
        max_file_descriptors: number;
        mem: {
          total_virtual_in_bytes: number;
        };
        open_file_descriptors: number;
        timestamp: number;
      };
      roles: string[];
      thread_pool: {
        [poolName: string]: {
          active: number;
          completed: number;
          largest: number;
          queue: number;
          rejected: number;
          threads: number;
        };
      };
      timestamp: number;
      transport_address: string;
    };
  };
}
