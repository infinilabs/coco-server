import type { FC } from 'react';
import type { LucideProps } from 'lucide-react';
import {
  Archive,
  ArrowUpRight,
  AudioLines,
  BookOpen,
  Bookmark,
  Bot,
  Box,
  BrushCleaning,
  Calendar,
  ChevronDown,
  ChevronRight,
  ChevronUp,
  Clock,
  Code,
  CornerDownLeft,
  Cpu,
  Database,
  File,
  FileText,
  Folder,
  Globe,
  Hash,
  Heart,
  Image,
  Layers,
  Lightbulb,
  Link,
  Mail,
  MessageCircle,
  Mic,
  Paperclip,
  Search,
  Sparkles,
  Star,
  Tag,
  User,
  Users,
  Video,
  Zap
} from 'lucide-react';

// Curated static registry. Data-driven icon names (document/suggestion icon
// fields) previously resolved by dynamically importing the whole lucide-react
// barrel — which forces every one of the ~1600 icons into the bundle because
// rollup must preserve the full namespace for a runtime lookup. This registry
// covers the names realistic data carries; unknown names fall back to a
// neutral document icon instead of rendering nothing.
const REGISTRY: Record<string, FC<LucideProps>> = {
  Archive,
  ArrowUpRight,
  AudioLines,
  BookOpen,
  Bookmark,
  Bot,
  Box,
  BrushCleaning,
  Calendar,
  ChevronDown,
  ChevronRight,
  ChevronUp,
  Clock,
  Code,
  CornerDownLeft,
  Cpu,
  Database,
  File,
  FileText,
  Folder,
  Globe,
  Hash,
  Heart,
  Image,
  Layers,
  Lightbulb,
  Link,
  Mail,
  MessageCircle,
  Mic,
  Paperclip,
  Search,
  Sparkles,
  Star,
  Tag,
  User,
  Users,
  Video,
  Zap
};

const KEBAB_TO_PASCAL = (key: string) =>
  key
    .split(/[-_\s]+/)
    .map(part => part.charAt(0).toUpperCase() + part.slice(1))
    .join('');

export function lookupLucideIcon(iconKey: string): FC<LucideProps> | undefined {
  const normalized = iconKey.trim();
  if (!normalized) return undefined;
  return REGISTRY[normalized] ?? REGISTRY[KEBAB_TO_PASCAL(normalized)];
}

export const DefaultLucideIcon = FileText;
